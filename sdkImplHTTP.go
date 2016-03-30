/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	tls "crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
)

//// requests honor this state, no need to pass in with every call
var bCloseConnections = true
var bDebugRequests = true
var bDebugResponses = true

func SetCloseConnectionMode(b bool) {
	bCloseConnections = b
}

func SetDebugRequestMode(b bool) {
	bDebugRequests = b
}

func SetDebugResponseMode(b bool) {
	bDebugResponses = b
}

func sdkLog(s string) { 	// formerly the gateway to glog.Info
	fmt.Println(s)
}

//// most funcs here return HttpError, which is an error

const ( // HttpError codes when the error occurred here, not in the remote call.  Hijacking the 000 range for this.
	HTTP_ERROR_UNKNOWN   = 0
	HTTP_ERROR_NOCREDS   = 1
	HTTP_ERROR_CLIENT    = 2
	HTTP_ERROR_NOREQUEST = 3
	HTTP_ERROR_JSON      = 4
)

type HttpError interface {
	Error() string // extends Error
	Code() int     // a real HTTP response code, or one of the 0xx codes above
	Chain() error
}

type implHttpError struct {
	errMessage string
	errCode    int
	errChain   error
}

func (e implHttpError) Error() string {
	return e.errMessage
}

func (e implHttpError) Code() int {
	return e.errCode
}

func (e implHttpError) Chain() error {
	return e.errChain
}

// should fail to compile, now that HttpError is an interface (hence pointer type) we should return &implHttpError
func makeError(msg string, code int, chain error) HttpError {
	if msg == "" { // msg required
		msg = "<HttpError, message text not available>"
	}

	if code == 0 {
		code = HTTP_ERROR_UNKNOWN
	}

	return implHttpError{
		errMessage: msg,
		errCode:    code,
		errChain:   chain, // often nil
	}
}

//// Credentials is returned from the login func, and used by everything else
type Credentials struct {
	Username      string
	Password      string // kept because we need reauth, especially when a token expires
	AccountAlias  string
	LocationAlias string // do we need this?
	BearerToken   string
}

func (obj *Credentials) GetUsername() string {
	return obj.Username
}

func (obj *Credentials) GetAccount() string {
	return obj.AccountAlias
}

func (obj *Credentials) GetLocation() string {
	return obj.LocationAlias
}

func (obj *Credentials) IsValid() bool {
	return (obj.AccountAlias != "") && (obj.BearerToken != "")
}

// and no GetBearerToken or GetPassword - keep them private within this file

func (obj *Credentials) ClearCredentials() { // creds object is useless after this
	obj.Username = ""
	obj.Password = ""
	obj.AccountAlias = ""
	obj.LocationAlias = ""
	obj.BearerToken = ""
}

func makeErrorOld(content string) error {
	if content == "" {
		content = "<error text not available>"
	}

	return errors.New("CLC API: " + content)
}

var dummyCreds = Credentials{Username: "dummy object passed by login proc and not used", Password: "no password here",
	AccountAlias: "invalid", LocationAlias: "invalid", BearerToken: "invalid"} // note dummyCreds.IsValid() is true

func GetCredentials(server, uri string, username, password string) (*Credentials, HttpError) {
	if (username == "") || (password == "") {
		return nil, makeError("username and/or password not provided", HTTP_ERROR_NOCREDS, nil)
	}

	body := fmt.Sprintf("{\"username\":\"%s\",\"password\":\"%s\"}", username, password)
	b := bytes.NewBufferString(body)

	authresp := AuthLoginResponseJSON{}

	err := invokeHTTP("POST", server, uri, &dummyCreds, b, &authresp)
	if err != nil {
		sdkLog("CLC failed to log in")
		return nil, err
	}

	sdkLog(fmt.Sprintf("assigning new token, do this:  export CLC_API_TOKEN=%s\n", authresp.BearerToken))
	sdkLog(fmt.Sprintf("also CLC_API_USERNAME=%s  CLC_API_ACCOUNT=%s  CLC_API_LOCATION=%s\n", authresp.Username, authresp.AccountAlias, authresp.LocationAlias))

	return &Credentials{
		Username:      authresp.Username,
		Password:      password,
		AccountAlias:  authresp.AccountAlias,
		LocationAlias: authresp.LocationAlias,
		BearerToken:   authresp.BearerToken,
	}, nil
}

func ReauthCredentials(creds *Credentials, server, uri string) error {
	creds.AccountAlias = ""
	creds.LocationAlias = ""
	creds.BearerToken = ""

	body := fmt.Sprintf("{\"username\":\"%s\",\"password\":\"%s\"}", creds.Username, creds.Password)
	b := bytes.NewBufferString(body)

	authresp := AuthLoginResponseJSON{}

	err := invokeHTTP("POST", server, uri, &dummyCreds, b, &authresp)
	if err != nil {
		return err
	}

	sdkLog(fmt.Sprintf("assigning new token, do this:  export CLC_API_TOKEN=%s\n", authresp.BearerToken))

	creds.AccountAlias = authresp.AccountAlias
	creds.LocationAlias = authresp.LocationAlias
	creds.BearerToken = authresp.BearerToken

	return nil
}

type AuthLoginRequestJSON struct { // actually this is unused, as we simply sprintf the string
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthLoginResponseJSON struct {
	Username      string   `json:"username"`
	AccountAlias  string   `json:"accountAlias"`
	LocationAlias string   `json:"locationAlias"`
	Roles         []string `json:"roles"`
	BearerToken   string   `json:"bearerToken"`
}

// no request message body sent.  Response body returned if ret is not nil
func simpleGET(server, uri string, creds *Credentials, ret interface{}) HttpError {
	return invokeHTTP("GET", server, uri, creds, nil, ret)
}

// no request message body sent.  Response body returned if ret is not nil
func simpleDELETE(server, uri string, creds *Credentials, ret interface{}) HttpError {
	return invokeHTTP("DELETE", server, uri, creds, nil, ret)
}

// body must be a json-annotated struct, and is marshalled into the request body
func marshalledPOST(server, uri string, creds *Credentials, body interface{}, ret interface{}) HttpError {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(body)
	if err != nil {
		return makeError("JSON marshalling failed", HTTP_ERROR_JSON, err)
	}

	return invokeHTTP("POST", server, uri, creds, b, ret)
}


// body must be a json-annotated struct, and is marshalled into the request body
func marshalledPUT(server, uri string, creds *Credentials, body interface{}, ret interface{}) HttpError {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(body)
	if err != nil {
		return makeError("JSON marshalling failed", HTTP_ERROR_JSON, err)
	}

	return invokeHTTP("PUT", server, uri, creds, b, ret)
}

// body is a JSON string, sent directly as the request body
func simplePOST(server, uri string, creds *Credentials, body string, ret interface{}) HttpError {
	b := bytes.NewBufferString(body)
	return invokeHTTP("POST", server, uri, creds, b, ret)
}

// method to be "GET", "POST", etc.
// server name "api.ctl.io" or "api.loadbalancer.ctl.io"
// uri always starts with /   (we assemble https://<server><uri>)
// creds required for anything except the login call
// body may be be nil
func invokeHTTP(method, server, uri string, creds *Credentials, body io.Reader, ret interface{}) HttpError {
	if (creds == nil) || !creds.IsValid() {
		return makeError("username and/or password not provided", HTTP_ERROR_NOCREDS, nil)
	}

	full_url := ("https://" + server + uri)
	req, err := http.NewRequest(method, full_url, body)
	if err != nil {
		return makeError("could not create HTTP request for "+full_url, HTTP_ERROR_NOREQUEST, err)
	} else if body != nil {
		req.Header.Add("Content-Type", "application/json") // incoming body to be a marshaled object already
	}

	req.Header.Add("Host", server) // the reason we take server and uri separately
	req.Header.Add("Accept", "application/json")

	isAuth := (creds == &dummyCreds)
	if !isAuth { // the login proc itself doesn't send an auth header
		req.Header.Add("Authorization", ("Bearer " + creds.BearerToken))
	}

	if bCloseConnections {
		req.Header.Add("Connection", "close")
	}

	if bDebugRequests {
		if isAuth {	// avoid writing username/password to the log
			sdkLog(fmt.Sprintf("auth request: %s", full_url))
		} else {
			v, _ := httputil.DumpRequestOut(req, true)
			sdkLog(string(v))
		}
	}

	// this should be the normal code
	//	resp,err := http.DefaultClient.Do(req)	// execute the call

	// instead, we have this which tolerates bad certs
	tlscfg := &tls.Config{InsecureSkipVerify: true} // true means to skip the verification
	transp := &http.Transport{TLSClientConfig: tlscfg}
	client := &http.Client{Transport: transp}
	resp, err := client.Do(req)
	// end of tolerating bad certs.  Do not keep this code - it allows MITM etc. attacks

	if bDebugResponses {
		vv, _ := httputil.DumpResponse(resp, true)
		sdkLog(string(vv))
	}

	if err != nil { // failed HTTP call
		return makeError("HTTP call failed", HTTP_ERROR_CLIENT, err) // chain the err
	}

	if resp.StatusCode == 401 { // Unauthorized.  Not a failure yet, perhaps we can reauth

		// nyi where to store auth server/uri?   In the creds object ?
		ReauthCredentials(creds, "api.ctl.io", "/v2/authentication/login")
		if creds.IsValid() {
			req.Header.Del("Authorization")
			req.Header.Add("Authorization", ("Bearer " + creds.BearerToken))

			resp, err = client.Do(req) // not :=
		}
	}

	if (resp.StatusCode < 200) || (resp.StatusCode >= 300) { // Q: do we care to distinguish the various 200-series codes?
		// stat := fmt.Sprintf("received HTTP response code %d\n", resp.StatusCode)

		if !bDebugRequests {
			sdkLog("dumping this request, after the fact")
			v, _ := httputil.DumpRequestOut(req, true)
			sdkLog(string(v))
		}

		if !bDebugResponses {
			vv, _ := httputil.DumpResponse(resp, true)
			sdkLog(string(vv))
		}

		return makeError("HTTP call failed", resp.StatusCode, nil)
	}

	if ret != nil { // permit methods without a response body, or calls that ignore the body and just look for status
		err = json.NewDecoder(resp.Body).Decode(ret)

		if err != nil {
			fmt.Println(fmt.Sprintf("JSON decode failed, err=%s", err.Error()))
			return makeError("JSON decode failed", HTTP_ERROR_JSON, err)
		}
	}

	return nil // success
}

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

// struct declarations provide the Go object model in which we present the API
type DataCenterName struct {
	DCID string
	Name string
}

type PoolNode struct {
	TargetIP   string // send traffic to this host
	TargetPort int    // at this port
}

type PoolDetails struct {
	PoolID string
	LBID   string // LB this pool belongs to

	IncomingPort int    // docs say 'the port on which incoming traffic will send requests', believed to mean 'where the LB is listening on the outside'
	Method       string // one of: 'roundrobin', 'leastconn'   Q: how to declare suitable constants for those
	Health       string // JSON object, as {unhealthyThreshold:2, healthyThreshold:2, intervalSeconds:5, targetPort:80}
	Persistence string // e.g. 'none'
	TimeoutMS   int64
	Mode        string // one of: 'tcp', 'http'

	Nodes []PoolNode
}

// Q: createLB to return this?  Or to just invoke inspectLB and return LBDetails?
type LoadBalancerCreationInfo struct {
	LBID        string // the ID should be enough.  This is only a struct so that we have a place to put new fields later if desired
	RequestTime int64  // per the server-side clock, whose synchronization with any other clock is unknown
}

type LoadBalancerDetails struct {
	LBID        string
	Name        string // unique within dc ?
	Description string
	PublicIP    string // omit privateIP, what would that mean?
	Pools       []PoolDetails
	Status      string // list of valid states?
	DataCenter  string
}

type LoadBalancerSummary struct {
	LBID        string
	Name        string
	Description string
	PublicIP    string
	DataCenter  string
}

type CenturyLinkClient interface {
	// authentication
	logout()
	hasCredentials() bool
	getUsername() string
	getAccountAlias() string

	// datacenter identification
	listAllDC() ([]DataCenterName, error)

	// load balancers
	createLB(datacenter string, name string, description string) (*LoadBalancerCreationInfo, error)
	deleteLB(dc, lbid string) (bool, error)
	inspectLB(dc, lbid string) (*LoadBalancerDetails, HttpError)
	listAllLB() ([]LoadBalancerSummary, error)

	inspectPool(dc, lbid, poolid string) (*PoolDetails, error)
	createPool(dc, lbid string, newpool *PoolDetails) (*PoolDetails, error) // send in newpool.PoolID=nil, the return will have it filled in
	updatePool(dc, lbid string, newpool *PoolDetails) (*PoolDetails, error) // send in newpool.PoolID, that's the pool whose details to update
	deletePool(dc, lbid string, poolID string) error
}

func ClientLogin(username, password string) (CenturyLinkClient, error) {
	return implClientLogin(username, password)
}

func ClientReload() (CenturyLinkClient, error) {
	return implClientFromEnv()
}

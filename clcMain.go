package main

import (
	"fmt"
	"bufio"
	"os"
	"strings"
	"strconv"
)


func main() {
	fmt.Printf("CenturyLinkCloud LBaaS client app\n")

	in := bufio.NewReader(os.Stdin)

	app := AppState {
		clc: nil,
	}

	for {  // infinite loop
		fmt.Printf("\n> ")	// prompt
		line, err := in.ReadString('\n')
		if err != nil {
			fmt.Printf("error reading stdin\n")
			break
		}

		parts := strings.Split(line, "\n")
		if len(parts) != 2 {
			fmt.Printf("error parsing stdin line ending\n")
			return
		}

		cmd := parts[0]
		if (cmd == "exit") || (cmd == "quit") {
			return
		}

		processInputLine(&app, parts[0])
	}
}

func processInputLine(app *AppState, in string) {
	
	parts := strings.Split(in, " ")
	nonnull_parts := make([]string, 0, len(parts))

	for idx := 0; idx < len(parts); idx++ {	// strings.Split leaves empty strings in place for consecutive spaces
		if parts[idx] != "" {
			nonnull_parts = append(nonnull_parts, parts[idx])
		}
	}

	if len(nonnull_parts) == 0 {
		return // just do nothing
	}

	cmd0 := ""	// crude, but this way the switch doesn't have nulls or array lengths to worry about
	cmd1 := ""
	cmd2 := ""
	cmd3 := ""
	cmd4 := ""

// golang fusses about this being unused
//	cmd5 := ""
//
//	if len(nonnull_parts) >= 6 {
//		cmd5 = nonnull_parts[5]
//	}

	if len(nonnull_parts) >= 5 {
		cmd4 = nonnull_parts[4]
	}

	if len(nonnull_parts) >= 4 {
		cmd3 = nonnull_parts[3]
	}

	if len(nonnull_parts) >= 3 {
		cmd2 = nonnull_parts[2]
	}

	if len(nonnull_parts) >= 2 {
		cmd1 = nonnull_parts[1]
	}

	if len(nonnull_parts) >= 1 {
		cmd0 = nonnull_parts[0]
	}

	if cmd0 == "help" {
		cmdHelp(nonnull_parts)

	} else if cmd0 == "args" {
		cmdArgs(nonnull_parts)

	} else if cmd0 == "auth" {
		if cmd1 == "login" {
			app.cmdAuthLogin(cmd2, cmd3) // "auth login user pass"
		} else if cmd1 == "env" {
			app.cmdAuthEnv()  // "auth env"
		} else if cmd1 == "logout" {
			app.cmdAuthLogout() // "auth logout"
		} else if cmd1 == "status" {
			app.cmdAuthStatus() // "auth status"
		} else {
			cmdUsage()
		}

	} else if cmd0 == "DC" {
		if cmd1 == "list" {
			app.cmdDatacenterList() // "DC list"
		} else {
			cmdUsage()
		}

	} else if cmd0 == "LB" {
		if cmd1 == "create" {
			app.cmdLoadbalancerCreate(cmd2, cmd3, cmd4) // "LB create dc name desc"
		} else if cmd1 == "delete" {
			app.cmdLoadbalancerDelete(cmd2, cmd3) // "LB delete dc lbid"
		} else if cmd1 == "details" {
			app.cmdLoadbalancerDetails(cmd2, cmd3) // "LB details dc lbid"
		} else if cmd1 == "list" {
			app.cmdLoadbalancerList() // "LB list"
		} else {
			cmdUsage()
		}

	} else if cmd0 == "pool" {
		if cmd1 == "create" {
			app.cmdPoolCreate(cmd2, cmd3, nonnull_parts) // "pool create dc lbid"
		} else if cmd1 == "update" {
			app.cmdPoolUpdate(cmd2, cmd3, cmd4, nonnull_parts) // "pool update dc lbid poolID"
		} else if cmd1 == "delete" {
			app.cmdPoolDelete(cmd2, cmd3, cmd4) // "pool delete dc lbid poolID"
		} else {
			cmdUsage()
		}

	} else {
		cmdUsage()
	}
}


func cmdUsage() {	// does not consider the args
	fmt.Printf("Usage:\n")
	fmt.Printf("\thelp\n")
	fmt.Printf("\texit\n")
	fmt.Printf("\tauth login username password\n")	
	fmt.Printf("\tauth env\n")
	fmt.Printf("\tauth logout\n")	
	fmt.Printf("\tauth status\n")
	fmt.Printf("\tDC list\n")	
	fmt.Printf("\tLB create DC name desc\n")	
	fmt.Printf("\tLB delete DC LBID\n")	
	fmt.Printf("\tLB details DC LBID\n")	
	fmt.Printf("\tLB list\n")	
	fmt.Printf("\tpool create DC LBID <pool details>\n")
	fmt.Printf("\tpool update DC LBID PoolID <pool details>\n")	
	fmt.Printf("\tpool delete DC LBID PoolID\n")	
}

func cmdArgs(parts []string) {	// accept any args, dump them out for debugging

	fmt.Printf("read %d parts\n", len(parts))
	for idx,s := range parts {
		fmt.Printf("part %d %q\n", idx, s)
	}
}

type AppState struct {
	clc CenturyLinkClient
}

func cmdHelp(args []string) {	//  args[0]="help"
	cmdUsage()	// start here
	fmt.Printf("No command-specific help available\n")
}

func (app *AppState) cmdAuthEnv() {		// wrapper that fetches user/pass from env

	if app.clc != nil {
		app.clc.logout()
		app.clc = nil
	}

	new_clc, err := ClientReload()
	if err != nil {
		fmt.Printf("could not log in: err=%s\n", err.Error())
		app.clc = nil
	} else {
		app.clc = new_clc
		fmt.Printf("logged in: user=%s, accountAlias=%s\n", app.clc.getUsername(), app.clc.getAccountAlias())
	}
}

func (app *AppState) cmdAuthLogin(argUsername string, argPassword string) {
	if (argUsername == "") || (argPassword == "") {
		cmdUsage()
		return
	}

	if app.clc != nil {
		app.clc.logout()
		app.clc = nil
	}

	new_clc, err := ClientLogin(argUsername, argPassword)
	if err != nil {
		fmt.Printf("could not log in: err=%s\n", err.Error())
		app.clc = nil
	} else {
		app.clc = new_clc
		fmt.Printf("logged in: user=%s, accountAlias=%s\n", app.clc.getUsername(), app.clc.getAccountAlias())
	}
}

func (app *AppState) cmdAuthLogout() {
	if app.clc != nil {
		user := app.clc.getUsername()
		app.clc.logout()		// nyi shouldn't logout return an error if the command cannot be executed?
		app.clc = nil
		fmt.Printf("user %s is logged out\n", user)
	} else {
		fmt.Printf("no user was logged in\n")
	}
}

func (app *AppState) cmdAuthStatus() {
	if app.clc != nil {
		fmt.Printf("logged in: user=%s, accountAlias=%s\n", app.clc.getUsername(), app.clc.getAccountAlias())
	} else {
		fmt.Printf("no user is logged in\n")
	}
}


func (app *AppState) cmdDatacenterList() {
	if app.clc == nil {
		fmt.Printf("no user is logged in\n")
		return
	}

	dclist, err := app.clc.listAllDC()
	if err != nil {
		fmt.Printf("remote call failed, err=%s\n", err.Error())
		return
	}

	for _,dc := range dclist {
		fmt.Printf("DC: id=%s, name=\"%s\"\n", dc.DCID, dc.Name)
	}
}

func (app *AppState) cmdLoadbalancerCreate(argDC string, argName string, argDesc string) {
	if app.clc == nil {
		fmt.Printf("no user is logged in\n")
		return
	}

	lbinf,err := app.clc.createLB(argDC, argName, argDesc)
	if err != nil {
		fmt.Printf("remote call failed, err=%s\n", err.Error())
		return
	}
	
	fmt.Printf("createLB status: lbid=%s \n", lbinf.LBID)
}

func (app *AppState) cmdLoadbalancerDelete(argDC string, argLBID string) {
	if app.clc == nil {
		fmt.Printf("no user is logged in\n")
		return
	}

	_,err := app.clc.deleteLB(argDC, argLBID)
	if err != nil {
		fmt.Printf("remote call failed, err=%s\n", err.Error())
		return
	}

	fmt.Printf("load balancer deleted\n")
}

func (app *AppState) cmdLoadbalancerDetails(argDC string, argLBID string) {
	if app.clc == nil {
		fmt.Printf("no user is logged in\n")
		return
	}

	lb,err := app.clc.inspectLB(argDC, argLBID)
	if err != nil {
		fmt.Printf("remote call failed, err=%s\n", err.Error())
		return
	}
	
	fmt.Printf("LB details: dc=%s, lbid=%s, status=%s, IP=%s \n",
		lb.DataCenter, lb.LBID, lb.Status, lb.PublicIP)
	fmt.Printf("  name=%s, description=%s \n", lb.Name, lb.Description)

	if len(lb.Pools) == 0 {
		fmt.Printf("  (no pools defined)\n")
	} else {
		for _,pool := range lb.Pools {
			printPoolDetails(&pool, "  ")
		}
	}
}

func (app *AppState) cmdLoadbalancerList() {
	if app.clc == nil {
		fmt.Printf("no user is logged in\n")
		return
	}

	lblist,err := app.clc.listAllLB()
	if err != nil {
		fmt.Printf("remote call failed, err=%s\n", err.Error())
		return
	}
	
	for _,lb := range lblist {	// we get LBSummary back
		fmt.Printf("LB: dc=%s, lbid=%s, name=\"%s\", desc=\"%s\",\n    ip=%s \n",
			lb.DataCenter, lb.LBID, lb.Name, lb.Description, lb.PublicIP)
	}
}


func (app *AppState) cmdPoolCreate(argDC string, argLBID string, args []string) {
	if app.clc == nil {
		fmt.Printf("no user is logged in\n")
		return
	}

	newpoolinfo, err := makePoolFromArgs(args, 4)
	if err != nil {
		fmt.Printf("invalid pool details: %s\n", err.Error())
		return
	}

	newpoolinfo.PoolID = ""
	newpoolinfo.LBID = argLBID
	
	pool,err := app.clc.createPool(argDC, argLBID, newpoolinfo)
	if err != nil {
		fmt.Printf("remote call failed, err=%s\n", err.Error())
		return
	}
	
	printPoolDetails(pool, "")
}


// nyi consider: give this app an env-like dictionary to reduce LBID cut&paste
// nyi consider: expand this app to do the whole rest of clc_sdk
// nyi consider: command-object dispatching

func printPoolDetails(pool *PoolDetails, inset string) {
	fmt.Printf("%spool: LBID:%s, PoolID:%s \n", inset, pool.LBID, pool.PoolID)
	fmt.Printf("%s  port:%d, method:%s, persistence:%s, timeout:%d, mode:%s \n", inset, 
		pool.IncomingPort, pool.Method, pool.Persistence, pool.TimeoutMS, pool.Mode)

	fmt.Printf("%s  health:%q\n", inset, pool.Health)  // JSON object, typically a longish string

	fmt.Printf("%s  nodes:[ ", inset)
	for _,node := range pool.Nodes {
		fmt.Printf("%s:%d ", node.TargetIP, node.TargetPort)
	}

	fmt.Printf("]\n")
}

func makePoolFromArgs(args []string, ignore int) (*PoolDetails, error) {
	pool := PoolDetails {	// install defaults
		PoolID:"",
		LBID:"",
		IncomingPort:8080,
		Method:"roundrobin",
		Health:"",
		Persistence:"none",
		TimeoutMS:1000,
		Mode:"tcp",
	}

	target_port := 8080

	for idx,s := range args {
		if idx < ignore {
			continue
		}

		if strings.HasPrefix(s, "port=") {
			s = strings.TrimPrefix(s, "port=")
			conv, e := strconv.Atoi(s)
			if e != nil {
				fmt.Printf("could not convert port number to integer: %s\n", s)
				return nil, fmt.Errorf("invalid pool details requested")
			}

			pool.IncomingPort = conv

		} else if strings.HasPrefix(s, "method=") {
			s = strings.TrimPrefix(s, "method=")
			pool.Method = s

		} else if strings.HasPrefix(s, "health=") {
			s = strings.TrimPrefix(s, "health=")
			pool.Health = s

		} else if strings.HasPrefix(s, "persistence=") {
			s = strings.TrimPrefix(s, "persistence=")
			pool.Persistence = s

		} else if strings.HasPrefix(s, "timeout=") {
			s = strings.TrimPrefix(s, "timeout=")
			conv, e := strconv.Atoi(s)
			if e != nil {
				fmt.Printf("could not convert timeout to integer: %s\n", s)
				return nil, fmt.Errorf("invalid pool details requested")
			}

			pool.TimeoutMS = int64(conv)

		} else if strings.HasPrefix(s, "mode=") {
			s = strings.TrimPrefix(s, "mode=")
			pool.Mode = s

		} else if strings.HasPrefix(s, "nodes=") {
			s = strings.TrimPrefix(s, "nodes=")
			parts := strings.Split(s, ",")	// comma-separated list with no spaces allowed

			nodes := make([]PoolNode, len(parts), len(parts))
			for idx,part := range parts {
				nodes[idx] = PoolNode {
					TargetIP:part,
					TargetPort:target_port,
				}
			}

			pool.Nodes = nodes

		} else if strings.HasPrefix(s, "target=") {
			s = strings.TrimPrefix(s, "target=")
			conv, e := strconv.Atoi(s)
			if e != nil {
				fmt.Printf("could not convert target port to integer: %s\n", s)
				return nil, fmt.Errorf("invalid pool details requested")
			}

			target_port = conv;

		} else {
			fmt.Printf("bad pool arg: %s \n", s)
			fmt.Printf("Pool Details fields: port, method, health, persistence, timeout, mode, nodes, target\n")
			return nil, fmt.Errorf("invalid pool details requested")
		}
	}

	fmt.Printf("parsed pool details from command line:\n")
	printPoolDetails(&pool, "    ")

	return &pool, nil
}

func (app *AppState) cmdPoolUpdate(argDC string, argLBID string, argPoolID string, args []string) {
	if app.clc == nil {
		fmt.Printf("no user is logged in\n")
		return
	}

	newpoolinfo, err := makePoolFromArgs(args, 5)
	if err != nil {
		fmt.Printf("invalid pool details: %s\n", err.Error())
		return 
	}

	newpoolinfo.PoolID = argPoolID
	newpoolinfo.LBID = argLBID
	
	pool,err := app.clc.updatePool(argDC,argLBID, newpoolinfo)
	if err != nil {
		fmt.Printf("remote call failed, err=%s\n", err.Error())
		return
	}
	
	printPoolDetails(pool, "")
}

func (app *AppState) cmdPoolDelete(argDC string, argLBID string, argPoolID string) {
	if app.clc == nil {
		fmt.Printf("no user is logged in\n")
		return
	}

	err := app.clc.deletePool(argDC, argLBID, argPoolID)
	if err != nil {
		fmt.Printf("remote call failed, err=%s\n", err.Error())
		return
	}

	fmt.Printf("pool deleted\n")
}


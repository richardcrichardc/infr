package main

import (
	"flag"
	"fmt"
	"infr/evilbootstrap"
	"infr/util"
	"os"
)

const hostsHelp = `Usage: infr hosts [options] [name]`

const infrDomain = "infr.tawherotech.nz"

var hostsRemove bool
var hostsAddStr, hostsAddPass string

type host struct {
	Name string
}

func hostsFlags(fs *flag.FlagSet) {
	fs.StringVar(&hostsAddStr, "a", "", "Add host 'name' to cluster, by doing an 'evil bootstrap' of host specified by this option")
	fs.StringVar(&hostsAddPass, "p", "", "Password for ...")
	fs.BoolVar(&hostsRemove, "r", false, "Remove host 'name' from cluster")
}

func hosts(args []string) {
	var name string

	hostsAdd := hostsAddStr != ""

	switch len(args) {
	case 0:
		if hostsAdd || hostsRemove {
			errorHelpExit("hosts", "That option requires 'name' to be specified.")
		}
	case 1:
		name = args[0]
		if hostsAdd && hostsRemove {
			errorHelpExit("hosts", "You cannot add and remove a host at the same time.")
		}
		if !hostsAdd && !hostsRemove {
			errorHelpExit("hosts", "Please specify an option so I know what to do with that host.")
		}
	default:
		errorHelpExit("hosts", "Too many arguments.")
	}

	if hostsAdd {
		hostsAddDo(name, hostsAddStr)
	} else if hostsRemove {
		hostsRemoveDo(name)
	} else {
		var hosts []host
		loadConfig("hosts", &hosts)
		for _, host := range hosts {
			fmt.Println(host.Name)
		}
	}
}

func hostsAddDo(name, target string) {
	var hosts []host
	var sshKeys string

	loadConfig("hosts", &hosts)
	loadConfig("keys", &sshKeys)

	if sshKeys == "" {
		errorHelpExit("keys", "No ssh keys configured. Use `infr keys -a` to add keys before adding hosts.")
	}

	for _, host := range hosts {
		if host.Name == name {
			errorExit("Host already exists: %s", name)
		}
	}

	input, err := util.Prompt("ARE YOU SURE YOU WANT TO REINSTALL THE OPERATING SYSTEM ON THE MACHINE AT " + target + "? (type YES to confirm) ")
	if err != nil {
		errorExit("%s", err.Error())
	}

	if input != "YES\n" {
		errorExit("ABORTING")
	}

	hosts = append(hosts, host{name})
	saveConfig("hosts", hosts)

	// evil bootstrap does a git checkout of ipxe in cwd, workdir is a good place for it
	cdWorkDir()

	err = evilbootstrap.Install(target, hostsAddPass, name, infrDomain, sshKeys)
	if err != nil {
		errorExit("Error during evil bootstrap: %s", err)
	}
}

func hostsRemoveDo(name string) {
	var hosts []host
	loadConfig("hosts", &hosts)

	var newHosts []host
	var removed bool

	for _, host := range hosts {
		if host.Name != name {
			newHosts = append(newHosts, host)
		} else {
			removed = true
		}
	}

	if !removed {
		fmt.Printf("Host not found: %s\n", name)
		os.Exit(1)
	}

	saveConfig("hosts", newHosts)
}

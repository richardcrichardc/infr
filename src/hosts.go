package main

import (
	"flag"
	"fmt"
	"infr/evilbootstrap"
	"infr/util"
	"os"
)

const hostsHelp = `Usage: infr hosts [subcommand] [args]

Manage hosts that containers are run on.
`

const infrDomain = "infr.tawherotech.nz"

var hostsRemove bool
var hostsAddStr, hostsAddPass string

type host struct {
	Name       string
	PublicIPv4 string
}

func hostsFlags(fs *flag.FlagSet) {
	fs.StringVar(&hostsAddStr, "a", "", "Add host 'name' to cluster, by doing an 'evil bootstrap' of host specified by this option")
	fs.StringVar(&hostsAddPass, "p", "", "Password for ...")
	fs.BoolVar(&hostsRemove, "r", false, "Remove host 'name' from cluster")
}

const hostsListHelp = `[list]

List all hosts.
`

func hostsListCmd(args []string) {
	if len(args) != 0 {
		errorHelpExit("hosts", "Too many arguments for 'list'.")
	}

	var hosts []host
	loadConfig("hosts", &hosts)
	for _, host := range hosts {
		fmt.Printf("%-15s %-15s", host.Name, host.PublicIPv4)
	}
}

const hostsAddHelp = `add [-p root-password] <name> <target IP address>

Add new host to cluster by sshing into root@<target IP address>, reformating the harddrive,
installing and configuring new operating system and other software.

THIS WILL DESTROY ALL DATA ON THE MACHINE AT <target IP address>. It is designed to be used with a
brand new VPS containing no data, USE ON AN EXISTING MACHINE AT YOUR OWN RISK.

This command uses the ssh key in $HOMEDIR/.ssh/id_rsa or the password provided by the -p flag to
authenticate with the target host.

`

func hostsAddFlags(fs *flag.FlagSet) {
	fs.StringVar(&hostsAddPass, "p", "", "Optional password for sshing into host for initial install.")
}

func hostsAddCmd(args []string) {
	if len(args) != 2 {
		errorHelpExit("hosts", "Wrong number of arguments for 'add'.")
	}

	name := args[0]
	publicIPv4 := args[1]

	var hosts []host
	var sshKeys string
	var lastPreseedURL string

	loadConfig("hosts", &hosts)
	loadConfig("keys", &sshKeys)
	loadConfig("lastPreseedURL", &lastPreseedURL)

	if sshKeys == "" {
		errorHelpExit("keys", "No ssh keys configured. Use `infr keys -a` to add keys before adding hosts.")
	}

	for _, host := range hosts {
		if host.Name == name {
			errorExit("Host already exists: %s", name)
		}
	}

	input, err := util.Prompt("ARE YOU SURE YOU WANT TO REINSTALL THE OPERATING SYSTEM ON THE MACHINE AT " + publicIPv4 + "? (type YES to confirm) ")
	if err != nil {
		errorExit("%s", err.Error())
	}

	if input != "YES\n" {
		errorExit("ABORTING")
	}

	hosts = append(hosts, host{Name: name, PublicIPv4: publicIPv4})
	saveConfig("hosts", hosts)

	// evil bootstrap does a git checkout of ipxe in cwd, workdir is a good place for it
	cdWorkDir()

	preseedURL, err := evilbootstrap.Install(publicIPv4, hostsAddPass, name, infrDomain, sshKeys, lastPreseedURL)

	if preseedURL != "" {
		saveConfig("lastPreseedURL", preseedURL)
	}

	if err != nil {
		errorExit("Error whilst reinstalling target: %s", err)
	}

}

const hostsRemoveHelp = `remove <name>

Remove named host from cluster.

At this stage the host is just removed from list of hosts.
`

func hostsRemoveCmd(args []string) {
	if len(args) != 1 {
		errorHelpExit("hosts", "Wrong number of arguments for 'remove'.")
	}

	name := args[0]

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

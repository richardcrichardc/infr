package main

import (
	"flag"
	"fmt"
	"infr/evilbootstrap"
	"infr/util"
)

func hostsCmd(args []string) {
	if len(args) == 0 {
		hostsListCmd(args)
	} else {
		switch args[0] {
		case "list":
			hostsListCmd(parseFlags(args, noFlags))
		case "add":
			hostsAddCmd(parseFlags(args, hostsAddFlags))
		default:
			errorExit("Invalid command: hosts %s", args[0])
		}
	}
}

func hostCmd(args []string) {
	if len(args) < 2 {
		errorExit("Not enough arguments for 'host'.")
	}

	h := findHost(args[0])
	args = args[1:]

	switch args[0] {
	case "reconfigure":
		hostsReconfigureCmd(h, parseFlags(args, hostsReconfigureFlags))
	case "remove":
		hostsRemoveCmd(h, parseFlags(args, noFlags))
	default:
		errorExit("Invalid command: %s", args[0])
	}
}

func hostsListCmd(args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'hosts [list]'.")
	}

	fmt.Printf("NAME            PUBLIC IP       PRIVATE IP\n")
	fmt.Printf("==========================================\n")
	for _, host := range config.Hosts {
		fmt.Printf("%-15s %-15s %-15s\n", host.Name, host.PublicIPv4, host.PrivateIPv4)
	}
}

//   hostsAdd flags
var hostsAddPass string

func hostsAddFlags(fs *flag.FlagSet) {
	fs.StringVar(&hostsAddPass, "p", "", "Optional password for sshing into host for initial install.")
}

func hostsAddCmd(args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'hosts add [-p <root-password>] <name> <target IP address>'.")
	}

	name := args[0]
	publicIPv4 := args[1]

	sshKeys := needKeys()
	infrDomain := needInfrDomain()

	for _, host := range config.Hosts {
		if host.Name == name {
			errorExit("Host already exists: %s", name)
		}
	}

	newHost := host{
		Name:        name,
		PublicIPv4:  publicIPv4,
		PrivateIPv4: vnetGetIP(),
	}

	input, err := util.Prompt("ARE YOU SURE YOU WANT TO REINSTALL THE OPERATING SYSTEM ON THE MACHINE AT " + publicIPv4 + "? (type YES to confirm) ")
	if err != nil {
		errorExit("%s", err.Error())
	}

	if input != "YES\n" {
		errorExit("ABORTING")
	}

	config.Hosts = append(config.Hosts, &newHost)
	saveConfig()

	// evil bootstrap does a git checkout of ipxe in cwd, workdir is a good place for it
	cdWorkDir()

	config.LastPreseedURL, err = evilbootstrap.Install(publicIPv4, hostsAddPass, name, infrDomain, sshKeys, config.LastPreseedURL)

	// Save preseed URL if evilbootstrap got that far, even if it otherwise failed
	if config.LastPreseedURL != "" {
		saveConfig()
	}

	if err != nil {
		errorExit("Error whilst reinstalling target: %s", err)
	}

	newHost.retrieveHostSSHPubKey()
	saveConfig()

	newHost.ConfigureNetwork()
	newHost.InstallSoftware()
	newHost.Configure()
	dnsFix()
}

func hostsRemoveCmd(toRemove *host, args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'host <name> remove'.")
	}

	var newHosts []*host

	for _, host := range config.Hosts {
		if toRemove != host {
			newHosts = append(newHosts, host)
		}
	}

	config.Hosts = newHosts
	saveConfig()

	dnsFix()
}

// hostsReconfigure Flags
var reconfigureNetwork, reinstallSoftware, retrieveHostSSHPubKey, notReconfigureHost bool

func hostsReconfigureFlags(fs *flag.FlagSet) {
	fs.BoolVar(&reconfigureNetwork, "n", false, "Reconfigure network on host.")
	fs.BoolVar(&reinstallSoftware, "s", false, "Reinstall software on host.")
	fs.BoolVar(&retrieveHostSSHPubKey, "k", false, "Retrieve hosts ssh public key.")
	fs.BoolVar(&notReconfigureHost, "R", false, "Don't reconfigure HAProxy etc.")
}

func hostsReconfigureCmd(h *host, args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'host <name> reconfigure'.")
	}

	if reconfigureNetwork {
		input, err := util.Prompt(`RECONFIGURING NETWORK ON HOST WILL BUMP ALL CONTAINERS OFF THE NETWORK (RESTART
HOST OR CONTAINERS TO REATTACH) AND IS ONLY NEEDED IF VNET IP CHANGES.
DO YOU WANT TO CONTINUE? (type YES to confirm) `)
		if err != nil {
			errorExit("%s", err.Error())
		}

		if input != "YES\n" {
			errorExit("ABORTING")
		}

		h.ConfigureNetwork()
	}

	if reinstallSoftware {
		h.InstallSoftware()
	}

	if retrieveHostSSHPubKey {
		h.retrieveHostSSHPubKey()
	}

	if !notReconfigureHost {
		h.Configure()
	}
}

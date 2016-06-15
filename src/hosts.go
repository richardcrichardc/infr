package main

import (
	"flag"
	"fmt"
	"os"
)

const hostsHelp = `Usage: infr hosts [options] [name]`

const infrDomain = "infr.tawherotech.nz"

var hostsAdd, hostsRemove bool

type host struct {
	Name string
}

func hostsFlags(fs *flag.FlagSet) {
	fs.BoolVar(&hostsAdd, "a", false, "Add host 'name' to cluster")
	fs.BoolVar(&hostsRemove, "r", false, "Remove host 'name' from cluster")
}

func hosts(args []string) {
	var name string

	switch len(args) {
	case 0:
		if hostsAdd || hostsRemove {
			errorHelpAndExit("hosts", "That option requires 'name' to be specified.")
		}
	case 1:
		name = args[0]
		if hostsAdd && hostsRemove {
			errorHelpAndExit("hosts", "You cannot add and remove a host at the same time.")
		}
		if !hostsAdd && !hostsRemove {
			errorHelpAndExit("hosts", "Please specify an option so I know what to do with that host.")
		}
	default:
		errorHelpAndExit("hosts", "Too many arguments.")
	}

	var hosts []host
	loadConfig("hosts", &hosts)

	if hostsAdd {
		for _, host := range hosts {
			if host.Name == name {
				fmt.Printf("Host already exists: %s\n", name)
				os.Exit(1)
			}
		}

		hosts = append(hosts, host{name})
	}

	if hostsRemove {
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

		hosts = newHosts
	}

	saveConfig("hosts", hosts)

	for _, host := range hosts {
		fmt.Println(host.Name)
	}
}

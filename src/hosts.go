package main

import (
	"flag"
	"fmt"
)

const hostsHelp = `Usage: infr hosts [-a address]`

const infrDomain = "infr.tawherotech.nz"

var hostsAdd, hostsRemove string

type host struct {
	Name string
}

func hostsFlags(fs *flag.FlagSet) {
	fs.StringVar(&hostsAdd, "a", "", "Add host to cluster")
	fs.StringVar(&hostsRemove, "r", "", "Remove host from cluster")
}

func hosts(args []string) {
	var hosts []host
	loadConfig("hosts", &hosts)

	if hostsAdd != "" {
		hosts = append(hosts, host{hostsAdd})
	}

	if hostsRemove != "" {
		var newHosts []host

		for _, host := range hosts {
			if host.Name != hostsRemove {
				newHosts = append(newHosts, host)
			}
		}

		hosts = newHosts
	}

	saveConfig("hosts", hosts)

	for _, host := range hosts {
		fmt.Println(host.Name)
	}
}

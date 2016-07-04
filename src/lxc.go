package main

import (
	"fmt"
)

const lxcHelp = `Usage: infr con [subcommand] [args]

Manage containers...
`

type lxc struct {
	Name        string
	Host        string
	PrivateIPv4 string
	Distro      string
	Release     string
}

const lxcListHelp = `[list]

List all containers.
`

func lxcListCmd(args []string) {
	if len(args) != 0 {
		errorHelpExit("lxc", "Too many arguments for 'list'.")
	}

	var lxcs []lxc
	loadConfig("lxcs", &lxcs)
	for _, lxc := range lxcs {
		fmt.Printf("%-15s %-15s", lxc.Name, lxc.Host, lxc.PrivateIPv4)
	}

	fmt.Println("next vnet ip:", vnetGetIP())
}

func lxcAddCmd(args []string) {
}

/*
sudo lxc-create -n zzz -B btrfs -t download -- -d ubuntu -r xenial -a amd64

# Template used to create this container: /usr/share/lxc/templates/lxc-download
# Parameters passed to the template: -d ubuntu -r xenial -a amd64
# For additional config options, please look at lxc.container.conf(5)

# Distribution configuration
lxc.include = /usr/share/lxc/config/ubuntu.common.conf
lxc.arch = x86_64

# Container specific configuration
lxc.rootfs = /var/lib/lxc/zzz/rootfs
lxc.utsname = zzz

# Network configuration
lxc.network.type = empty

----

# Template used to create this container: /usr/share/lxc/templates/lxc-download
# Parameters passed to the template:
# For additional config options, please look at lxc.container.conf(5)

# Distribution configuration
lxc.include = /usr/share/lxc/config/ubuntu.common.conf
lxc.arch = x86_64

# Container specific configuration
lxc.rootfs = /var/lib/lxc/max/rootfs
lxc.utsname = max

# Network configuration
lxc.network.type = veth
lxc.network.flags = up
lxc.network.link = brvm
lxc.network.hwaddr = 52:54:00:59:70:f2
lxc.start.auto = 1

----

# Template used to create this container: /usr/share/lxc/templates/lxc-debian
# Parameters passed to the template: -r jessie
# For additional config options, please look at lxc.container.conf(5)
lxc.network.type = empty
lxc.rootfs = /var/lib/lxc/castle/rootfs

# Common configuration
lxc.include = /usr/share/lxc/config/debian.common.conf

# Container specific configuration
lxc.mount = /var/lib/lxc/castle/fstab
lxc.utsname = castle
lxc.arch = amd64
lxc.autodev = 1
lxc.kmsg = 0
lxc.network.type = veth
lxc.network.flags = up
lxc.network.link = brvm
lxc.network.hwaddr = 52:54:00:59:be:ef
lxc.start.auto = 1


*/

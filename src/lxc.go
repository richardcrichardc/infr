package main

import (
	"fmt"
	"os"
	"strings"
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
		fmt.Printf("%-15s %-15s %-15s\n", lxc.Name, lxc.Host, lxc.PrivateIPv4)
	}
}

const lxcAddHelp = `add <name> <distro> <release> <host>

Create container called <name> running <distro> <release> on <host>.
`

func lxcAddCmd(args []string) {
	if len(args) != 4 {
		errorHelpExit("lxc", "Wrong number of arguments for 'add'.")
	}

	name := strings.ToLower(args[0])
	distro := strings.ToLower(args[1])
	release := strings.ToLower(args[2])
	hostname := strings.ToLower(args[3])

	var lxcs []lxc
	loadConfig("lxcs", &lxcs)
	for _, l := range lxcs {
		if l.Name == name {
			errorExit("Lxc already exists: %s", name)
		}
	}

	_ = findHost(hostname)

	newLxc := lxc{
		Name:        name,
		Host:        hostname,
		PrivateIPv4: vnetGetIP(),
		Distro:      distro,
		Release:     release}

	lxcs = append(lxcs, newLxc)

	saveConfig("lxcs", lxcs)

	newLxc.Create()
	dnsFix()
}

const lxcRemoveHelp = `remove <name>

Remove container from cluster.

At this stage the host is just removed from list of containers.
`

func lxcRemoveCmd(args []string) {
	if len(args) != 1 {
		errorHelpExit("lxc", "Wrong number of arguments for 'remove'.")
	}

	name := strings.ToLower(args[0])

	var lxcs []lxc
	loadConfig("lxcs", &lxcs)

	var newLxcs []lxc
	var removed bool

	for _, lxc := range lxcs {
		if lxc.Name != name {
			newLxcs = append(newLxcs, lxc)
		} else {
			removed = true
		}
	}

	if !removed {
		fmt.Printf("Lxc not found: %s\n", name)
		os.Exit(1)
	}

	saveConfig("lxcs", newLxcs)

	dnsFix()
}

func findLxc(name string) *lxc {
	var lxcs []lxc
	loadConfig("lxcs", &lxcs)

	for _, l := range lxcs {
		if l.Name == name {
			return &l
		}
	}

	errorExit("Unknown lxc: %s", name)
	return nil
}

func (l *lxc) Create() {
	host := findHost(l.Host)
	maskSize, _ := vnetNetwork().Mask.Size()

	data := lxcCreateData{
		lxc:             l,
		NetworkMaskSize: maskSize,
		GatewayIPv4:     host.BridgeIPv4}

	host.RunScript(createLxcScript, data, true, true)
}

type lxcCreateData struct {
	*lxc
	NetworkMaskSize int
	GatewayIPv4     string
}

const createLxcScript = `
# This script is idempotent

# echo commands and exit on error
set -v -e

lxc-create -n {{.Name}} -B btrfs -t download -- -d {{.Distro}} -r {{.Release}} -a amd64

cat <<'EOF' | confedit /var/lib/lxc/{{.Name}}/config
lxc.network.type = veth
lxc.network.flags = up
lxc.network.link = br0
lxc.network.name = eth0
lxc.network.ipv4 = {{.PrivateIPv4}}/{{.NetworkMaskSize}}
lxc.network.ipv4.gateway = {{.GatewayIPv4}}
lxc.start.auto = 1
EOF

lxc-start -d -n {{.Name}}
`

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

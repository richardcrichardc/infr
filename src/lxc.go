package main

import (
	"fmt"
	"net"
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
			lxc.Remove()
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

	data := lxcCreateData{
		lxc:                l,
		PrivateNetwork:     vnetNetwork(),
		PrivateNetworkMask: net.IP(vnetNetwork().Mask),
		GatewayIPv4:        host.BridgeIPv4,
		SSHKeys:            needKeys()}

	host.RunScript(createLxcScript, data, true, true)
}

type lxcCreateData struct {
	*lxc
	PrivateNetwork     *net.IPNet
	PrivateNetworkMask net.IP
	GatewayIPv4        string
	SSHKeys            string
}

const createLxcScript = `
# echo commands and exit on error
set -v -e

lxc-create -n {{.Name}} -B btrfs -t download -- -d {{.Distro}} -r {{.Release}} -a amd64

cat <<'EOF' | confedit /var/lib/lxc/{{.Name}}/config
lxc.network.type = veth
lxc.network.flags = up
lxc.network.link = br0
lxc.start.auto = 1
EOF

lxc-start -d -n {{.Name}}
lxc-wait -n {{.Name}} -s RUNNING

cat <<'EOF' | lxc-attach -n {{.Name}}

cat <<'EOG' > /etc/network/interfaces
# AUTOMATICALLY GENERATED - DO NOT EDIT

# This file describes the network interfaces available on your system
# and how to activate them. For more information, see interfaces(5).

source /etc/network/interfaces.d/*

# The loopback network interface
auto lo
iface lo inet loopback

# The primary network interface
auto eth0
iface eth0 inet static
    address {{.PrivateIPv4}}
    netmask {{.PrivateNetworkMask}}
    gateway {{.GatewayIPv4}}
    dns-nameserver 8.8.8.8
EOG

	ifdown -a
	ifup -a
	apt-get -y install openssh-server

	adduser --disabled-password --gecos "" manager

	mkdir /home/manager/.ssh
	chmod u=rwx /home/manager/.ssh
	echo "{{.SSHKeys}}" > /home/manager/.ssh/authorized_keys
	chmod u=rw /home/manager/.ssh/authorized_keys

	adduser manager sudo
	# allow sudo without password (manager has not got one)
	sed -i 's/sudo[[:space:]]*ALL=(ALL:ALL) ALL/sudo ALL=(ALL:ALL) NOPASSWD:ALL/' /etc/sudoers
EOF

`

func (l *lxc) Remove() {
	host := findHost(l.Host)
	host.RunScript(removeLxcScript, l, true, true)
}

const removeLxcScript = `
# echo commands and exit on error
set -v -e

lxc-destroy -f -n {{.Name}}
`

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

type httpAction int

const (
	HTTPNONE = iota
	HTTPFORWARD
	HTTPREDIRECT
)

type httpsAction int

const (
	HTTPSNONE = iota
	HTTPSTERMINATE
)

type lxc struct {
	Name        string
	Host        string
	PrivateIPv4 string
	Distro      string
	Release     string
	Aliases     []string
	Http        httpAction
	Https       httpsAction
}

func lxcCmd(args []string) {
	if len(args) == 0 {
		lxcListCmd(args)
	} else {
		switch args[0] {
		case "list":
			lxcListCmd(parseFlags(args, noFlags))
		case "add":
			lxcAddCmd(parseFlags(args, noFlags))
		case "remove":
			lxcRemoveCmd(parseFlags(args, noFlags))
		case "show":
			lxcShowCmd(parseFlags(args, noFlags))
		case "alias-add":
			lxcAliasAddCmd(parseFlags(args, noFlags))
		case "alias-remove":
			lxcAliasRemoveCmd(parseFlags(args, noFlags))
		case "http":
			lxcHttpCmd(parseFlags(args, noFlags))
		case "https":
			lxcHttpsCmd(parseFlags(args, noFlags))

		default:
			errorExit("Invalid command: %s", args[0])
		}
	}
}

const lxcListHelp = `[list]

List all containers.
`

func lxcListCmd(args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'list'.")
	}

	for _, lxc := range config.Lxcs {
		fmt.Printf("%-15s %-15s %-15s\n", lxc.Name, lxc.Host, lxc.PrivateIPv4)
	}
}

const lxcAddHelp = `add <name> <distro> <release> <host>

Create container called <name> running <distro> <release> on <host>.
`

func lxcAddCmd(args []string) {
	if len(args) != 4 {
		errorExit("Wrong number of arguments for 'add'.")
	}

	name := strings.ToLower(args[0])
	distro := strings.ToLower(args[1])
	release := strings.ToLower(args[2])
	hostname := strings.ToLower(args[3])

	for _, l := range config.Lxcs {
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

	config.Lxcs = append(config.Lxcs, newLxc)
	saveConfig()

	newLxc.Create()
	dnsFix()
}

const lxcRemoveHelp = `remove <name>

Remove container from cluster.

At this stage the host is just removed from list of containers.
`

func lxcRemoveCmd(args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'remove'.")
	}

	name := strings.ToLower(args[0])

	var newLxcs []lxc
	var removed bool

	for _, lxc := range config.Lxcs {
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

	config.Lxcs = newLxcs
	saveConfig()

	dnsFix()
}

func lxcShowCmd(args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'show'.")
	}

	lxc := findLxc(args[0])

	fmt.Printf("Name:     %s\n", lxc.Name)
	fmt.Printf("Host:     %s\n", lxc.Host)
	fmt.Printf("Distro:   %s %s\n", lxc.Distro, lxc.Release)
	fmt.Printf("Aliases:  %s\n", strings.Join(lxc.Aliases, ", "))
	fmt.Printf("HTTP: 	  %s\n", httpActionString(lxc.Http))
	fmt.Printf("HTTPS:    %s\n", httpsActionString(lxc.Https))
}

func httpActionString(a httpAction) string {
	switch a {
	case HTTPNONE:
		return "NONE"
	case HTTPFORWARD:
		return "FORWARD"
	case HTTPREDIRECT:
		return "REDIRECT"
	default:
		panic("Unknown httpAction")
	}
}

func httpsActionString(a httpsAction) string {
	switch a {
	case HTTPSNONE:
		return "NONE"
	case HTTPSTERMINATE:
		return "TERMINATE"
	default:
		panic("Unknown httpAction")
	}
}

func lxcAliasAddCmd(args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'alias-add'.")
	}

	lxc := findLxc(args[0])
	alias := strings.ToLower(args[1])

	lxc.Aliases = uniqueStrings(append(lxc.Aliases, alias))
	saveConfig()
	lxc.FindHost().Configure()
}

func lxcAliasRemoveCmd(args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'alias-remove'.")
	}

	lxc := findLxc(args[0])
	alias := strings.ToLower(args[1])

	lxc.Aliases = removeStrings(lxc.Aliases, []string{alias})
	saveConfig()
	lxc.FindHost().Configure()
}

func lxcHttpCmd(args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'http'.")
	}

	lxc := findLxc(args[0])
	option := strings.ToUpper(args[1])

	switch option {
	case "NONE":
		lxc.Http = HTTPNONE
	case "FORWARD":
		lxc.Http = HTTPFORWARD
	case "REDIRECT":
		lxc.Http = HTTPREDIRECT
	default:
		errorExit("Invalid option, please specify: NONE, FORWARD or REDIRECT")
	}

	saveConfig()
	lxc.FindHost().Configure()
}

func lxcHttpsCmd(args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'https'.")
	}

	lxc := findLxc(args[0])
	option := strings.ToUpper(args[1])

	switch option {
	case "NONE":
		lxc.Https = HTTPSNONE
	case "TERMINATE":
		lxc.Https = HTTPSTERMINATE
	default:
		errorExit("Invalid option, please specify: NONE or TERMINATE")
	}

	saveConfig()
	lxc.FindHost().Configure()
}

func findLxc(name string) *lxc {
	for i, l := range config.Lxcs {
		if l.Name == name {
			return &config.Lxcs[i]
		}
	}

	errorExit("Unknown lxc: %s", name)
	return nil
}

func (l *lxc) FindHost() *host {
	return findHost(l.Host)
}

func (l *lxc) Create() {
	host := l.FindHost()

	data := lxcCreateData{
		lxc:                l,
		PrivateNetwork:     vnetNetwork(),
		PrivateNetworkMask: net.IP(vnetNetwork().Mask),
		GatewayIPv4:        host.PrivateIPv4,
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

func (l *lxc) HttpBackend() string {
	switch l.Http {
	case HTTPNONE:
		return ""
	case HTTPFORWARD:
		return l.Name + "_http"
	case HTTPREDIRECT:
		return "redirect_https"
	default:
		panic("Unexpected httpAction")
	}
}

func (l *lxc) HttpsBackend() string {
	switch l.Https {
	case HTTPSNONE:
		return ""
	case HTTPSTERMINATE:
		return l.Name + "_http"
	default:
		panic("Unexpected httpsAction")
	}
}

func (l *lxc) FQDN() string {
	return l.Name + "." + needInfrDomain()
}

package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

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

type tcpForward struct {
	HostPort int
	LxcPort  int
}

type lxc struct {
	Name        string
	Host        string
	PrivateIPv4 string
	Distro      string
	Release     string
	Aliases     []string
	Http        httpAction
	Https       httpsAction
	HttpPort    int
	TCPForwards []*tcpForward
}

func lxcsCmd(args []string) {
	if len(args) == 0 {
		lxcListCmd(args)
	} else {
		switch args[0] {
		case "list":
			lxcListCmd(parseFlags(args, noFlags))
		case "add":
			lxcAddCmd(parseFlags(args, noFlags))
		default:
			errorExit("Invalid command: %s", args[0])
		}
	}
}

func lxcCmd(args []string) {
	if len(args) < 1 {
		errorExit("Not enough arguments for 'lxc'.")
	}

	l := findLxc(args[0])
	args = args[1:]

	if len(args) == 0 {
		lxcShowCmd(l, args)
	} else {
		switch args[0] {
		case "remove":
			lxcRemoveCmd(l, parseFlags(args, noFlags))
		case "show":
			lxcShowCmd(l, parseFlags(args, noFlags))
		case "add-alias":
			lxcAddAliasCmd(l, parseFlags(args, noFlags))
		case "remove-alias":
			lxcRemoveAliasCmd(l, parseFlags(args, noFlags))
		case "http":
			lxcHttpCmd(l, parseFlags(args, noFlags))
		case "https":
			lxcHttpsCmd(l, parseFlags(args, noFlags))
		case "http-port":
			lxcHttpPortCmd(l, parseFlags(args, noFlags))
		case "add-tcp-forward":
			lxcAddTcpForwardCmd(l, parseFlags(args, noFlags))
		case "remove-tcp-forward":
			lxcRemoveTcpForwardCmd(l, parseFlags(args, noFlags))

		default:
			errorExit("Invalid command: %s", args[0])
		}
	}
}

func lxcListCmd(args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'lxcs [list]'.")
	}

	for _, lxc := range config.Lxcs {
		fmt.Printf("%-15s %-15s %-15s\n", lxc.Name, lxc.Host, lxc.PrivateIPv4)
	}
}

func lxcAddCmd(args []string) {
	if len(args) != 4 {
		errorExit("Wrong number of arguments for 'lxcs add'.")
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
		Release:     release,
		HttpPort:    80}

	config.Lxcs = append(config.Lxcs, &newLxc)
	saveConfig()

	newLxc.Create()
	dnsFix()
}

func lxcRemoveCmd(toRemove *lxc, args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'lxc <name> remove'.")
	}

	var newLxcs []*lxc

	for _, lxc := range config.Lxcs {
		if toRemove != lxc {
			newLxcs = append(newLxcs, lxc)
		}
	}

	toRemove.Remove()

	config.Lxcs = newLxcs
	saveConfig()

	dnsFix()
}

func (l *lxc) FQDN() string {
	return l.Name + "." + infrDomain()
}

func (l *lxc) VnetFQDN() string {
	return l.Name + "." + vnetDomain()
}

func (l *lxc) TCPForwardsString() string {
	var rules []string

	for _, rule := range l.TCPForwards {
		rules = append(rules, fmt.Sprintf("%d=>%d", rule.HostPort, rule.LxcPort))
	}

	return strings.Join(rules, ", ")
}

func lxcShowCmd(l *lxc, args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'lxc <name> show'.")
	}

	fmt.Printf("Name:          %s\n", l.Name)
	fmt.Printf("Host:          %s\n", l.Host)
	fmt.Printf("Distro:        %s %s\n", l.Distro, l.Release)
	fmt.Printf("Aliases:       %s\n", strings.Join(l.Aliases, ", "))
	fmt.Printf("HTTP: 	       %s\n", httpActionString(l.Http))
	fmt.Printf("HTTPS:         %s\n", httpsActionString(l.Https))
	fmt.Printf("LXC Http Port: %d\n", l.HttpPort)
	fmt.Printf("TCP Forwards:  %s\n", l.TCPForwardsString())
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

func lxcAddAliasCmd(l *lxc, args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'lxc <name> add-alias <alias>'.")
	}

	alias := strings.ToLower(args[0])

	l.Aliases = uniqueStrings(append(l.Aliases, alias))
	saveConfig()
	l.FindHost().Configure()
}

func lxcRemoveAliasCmd(l *lxc, args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'lxc <name> remove-alias <alias>'.")
	}

	alias := strings.ToLower(args[0])

	l.Aliases = removeStrings(l.Aliases, []string{alias})
	saveConfig()
	l.FindHost().Configure()
}

func lxcHttpCmd(l *lxc, args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'lxc <name> http NONE|FORWARD|REDIRECT'.")
	}

	option := strings.ToUpper(args[0])

	switch option {
	case "NONE":
		l.Http = HTTPNONE
	case "FORWARD":
		l.Http = HTTPFORWARD
	case "REDIRECT":
		l.Http = HTTPREDIRECT
	default:
		errorExit("Invalid option, please specify: NONE, FORWARD or REDIRECT")
	}

	saveConfig()
	l.FindHost().Configure()
}

func lxcHttpsCmd(l *lxc, args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'lxc <name> https NONE|TERMINATE'.")
	}

	option := strings.ToUpper(args[0])

	switch option {
	case "NONE":
		l.Https = HTTPSNONE
	case "TERMINATE":
		l.Https = HTTPSTERMINATE
	default:
		errorExit("Invalid option, please specify: NONE or TERMINATE")
	}

	saveConfig()
	l.FindHost().Configure()
}

func lxcHttpPortCmd(l *lxc, args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'lxc <name> http-port <port>'.")
	}

	port, _ := strconv.Atoi(args[0]) // returns 0 or maxint on parse error

	if port < 1 || port > 65535 {
		errorExit("Invalid port, please specify an integer between 1 and 65535.")
	}

	l.HttpPort = port

	saveConfig()
	l.FindHost().Configure()
}

func lxcAddTcpForwardCmd(l *lxc, args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'lxc <name> add-tcp-forward <host-port> <lxc-port>'.")
	}

	rule := &tcpForward{}

	rule.HostPort, _ = strconv.Atoi(args[0]) // returns 0 or maxint on parse error
	rule.LxcPort, _ = strconv.Atoi(args[1])  // returns 0 or maxint on parse error

	if rule.HostPort < 1 || rule.HostPort > 65535 || rule.LxcPort < 1 || rule.LxcPort > 65535 {
		errorExit("Invalid port, please specify integers between 1 and 65535.")
	}

	if rule.HostPort == 22 || rule.HostPort == 80 || rule.HostPort == 443 {
		errorExit("Port %d on the host is reserved for use by the host.", rule.HostPort)
	}

	h := l.FindHost()

	for _, otherLxc := range h.AllLxcs() {
		for _, otherRule := range otherLxc.TCPForwards {
			if rule.HostPort == otherRule.HostPort {
				errorExit("Port %d on %s is already being forwarded to the lxc %s.", rule.HostPort, h.Name, otherLxc.Name)
			}
		}
	}

	l.TCPForwards = append(l.TCPForwards, rule)

	saveConfig()
	h.Configure()
}

func lxcRemoveTcpForwardCmd(l *lxc, args []string) {
	if len(args) != 2 {
		errorExit("Wrong number of arguments for 'lxc <name> remove-tcp-forward <host-port> <lxc-port>'.")
	}

	rule := tcpForward{}

	rule.HostPort, _ = strconv.Atoi(args[0]) // returns 0 or maxint on parse error
	rule.LxcPort, _ = strconv.Atoi(args[1])  // returns 0 or maxint on parse error

	if rule.HostPort < 1 || rule.HostPort > 65535 || rule.LxcPort < 1 || rule.LxcPort > 65535 {
		errorExit("Invalid port, please specify integers between 1 and 65535.")
	}

	var newForwards []*tcpForward
	found := false
	for _, otherRule := range l.TCPForwards {
		if rule != *otherRule {
			newForwards = append(newForwards, otherRule)
		} else {
			found = true
		}
	}

	if !found {
		errorExit("There is no forwarding rule %d=>%d on %s.", rule.HostPort, rule.LxcPort, l.Name)
	}

	l.TCPForwards = newForwards

	saveConfig()
	h := l.FindHost()
	h.Configure()
}

func findLxc(name string) *lxc {
	for i, l := range config.Lxcs {
		if l.Name == name {
			return config.Lxcs[i]
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

	host.SudoScript(createLxcScript, data)
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
	host.SudoScript(removeLxcScript, l)
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

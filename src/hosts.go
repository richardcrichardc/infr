package main

import (
	"bytes"
	"flag"
	"fmt"
	"infr/easyssh"
	"infr/evilbootstrap"
	"infr/util"
	"net"
	"os"
	"text/template"
)

const hostsHelp = `Usage: infr hosts [subcommand] [args]

Manage hosts that containers are run on.
`

var hostsRemove bool
var hostsAddStr, hostsAddPass string

type host struct {
	Name        string
	PublicIPv4  string
	PrivateIPv4 string
}

const hostsListHelp = `[list]

List all hosts.
`

func hostsListCmd(args []string) {
	if len(args) != 0 {
		errorHelpExit("hosts", "Too many arguments for 'list'.")
	}

	fmt.Printf("NAME            PUBLIC IP       PRIVATE IP\n")
	fmt.Printf("==========================================\n")
	for _, host := range config.Hosts {
		fmt.Printf("%-15s %-15s %-15s\n", host.Name, host.PublicIPv4, host.PrivateIPv4)
	}
}

const hostsAddHelp = `add [-p root-password] <name> <target IP address>

Add new host to cluster by sshing into root@<target IP address>, reformating the harddrive,
installing and configuring new operating system and other software.

THIS WILL DESTROY ALL DATA ON THE MACHINE AT <target IP address>. It is designed to be used with a
brand new VPS containing no data, USE ON AN EXISTING MACHINE AT YOUR OWN RISK.

This command uses the ssh key in $HOMEDIR/.ssh/id_rsa or the password provided by the -p flag to
authenticate with the target host.

Before using this command you need to use:
infr keys add <keyfile> -- to specify ssh keys to be installed on new host.
infr config set infrDomain <domain> -- to specify domain that will be appended to <name> to create hosts FQDN.

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

	config.Hosts = append(config.Hosts, newHost)
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

	newHost.ConfigureNetwork()
	newHost.InstallSoftware()
	newHost.Configure()
	dnsFix()
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

	var newHosts []host
	var removed bool

	for _, host := range config.Hosts {
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

	config.Hosts = newHosts
	saveConfig()

	dnsFix()
}

func hostsReconfigureCmd(args []string) {
	if len(args) != 1 {
		errorHelpExit("hosts", "Wrong number of arguments for 'reconfigure'.")
	}

	host := findHost(args[0])

	// don't reconfigure network as that knocks all the containers off the bridge
	host.InstallSoftware()
	host.Configure()
}

func hostsReconfigureNetworkCmd(args []string) {
	if len(args) != 1 {
		errorHelpExit("hosts", "Wrong number of arguments for 'reconfigure'.")
	}

	host := findHost(args[0])

	input, err := util.Prompt(`RECONFIGURING NETWORK ON HOST WILL BUMP ALL CONTAINERS OFF THE NETWORK (RESTART
HOST OR CONTAINERS TO REATTACH) AND IS ONLY NEEDED IF VNET IP CHANGES.
DO YOU WANT TO CONTINUE? (type YES to confirm) `)
	if err != nil {
		errorExit("%s", err.Error())
	}

	if input != "YES\n" {
		errorExit("ABORTING")
	}

	host.ConfigureNetwork()
}

func findHost(name string) *host {
	for _, host := range config.Hosts {
		if host.Name == name {
			return &host
		}
	}

	errorExit("Unknown host: %s", name)
	return nil
}

func (h *host) RunScript(scriptTmpl string, data interface{}, echo, sudo bool) {
	var script bytes.Buffer

	tmpl := template.Must(template.New("script").Parse(scriptTmpl))
	err := tmpl.Execute(&script, data)
	if err != nil {
		errorExit("Error executing script template: %s", err)
	}

	ssh := &easyssh.MakeConfig{
		User:   "manager",
		Server: h.PublicIPv4,
		Key:    "/.ssh/id_rsa",
		Port:   "22",
	}

	fmt.Printf("Running script on remote host: %s\n", h.Name)
	err = ssh.RunScript(script.String(), echo, sudo)
	if err != nil {
		errorExit("Error running remote script: %s", err)
	}
}

func (h *host) InstallSoftware() {
	h.RunScript(installSoftwareScript, h, true, true)
}

const installSoftwareScript = `
# This script is idempotent

# echo commands and exit on error
set -v -e

# install various packages
apt-get -y install lxc bridge-utils

# install confedit script used by this and other scripts
cat << EOF > /usr/local/bin/confedit
#!/usr/bin/python3

import sys
from collections import OrderedDict


def usage():
    print("""Usage: confscriptedit <dest-file>

Config file editor merges the config script provided on stdin into <dest-file>,
the result is saved to <dest-file>.

Config scripts consist of:
# comments

# ^^^ empty lines ^^^^, and
key = value-pairs

Merging occurs by replacing key value-pairs, matching by key, in <dest-file>
then appending all remaining items. Items specified with a blank value are
removed from <dest-file>. All other lines in <dest-file> are copied as is.
""")
    exit(1)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        usage()

    try:
        # read in destfile
        dest = sys.argv[1]
        f = open(dest)
        lines = f.read().splitlines()
        f.close()

        # read stdin into dict
        changes = OrderedDict()
        for line in sys.stdin.read().splitlines():
            if line.strip() == "" or line[0] == "#":
                continue

            key, sep, value = line.partition("=")
            if sep == "":
                print("Cannot understand input:", key)
                exit(1)

            changes[key.strip()] = value.strip()

        # write out original file making substitutions
        changed = set()
        f = open(dest, "w")
        for line in lines:
            key, sep, value = line.partition("=")
            stripped_key = key.strip()
            if sep == "" or (key and key[0] == "#") or stripped_key not in changes:
                f.write("%s\n" % line)
            else:
                f.write("%s = %s\n" % (stripped_key, changes[stripped_key]))
                changed.add(stripped_key)

        # write out new config items
        for key, value in changes.items():
            if key not in changed:
                f.write("%s = %s\n" % (key, value))

        f.close()
        exit(0)

    except IOError as e:
        print(e)
        exit(1)
EOF
chmod +x /usr/local/bin/confedit

# enable IP forwarding so nat from private network works
cat <<'EOF' | confedit /etc/sysctl.conf
net.ipv4.ip_forward = 1
EOF
sysctl --system

# install zerotier one
wget -O - https://install.zerotier.com/ | bash
`

func (h *host) Configure() {
	conf := hostConfigData{
		host:              h,
		ZerotierNetworkId: generalConfig("vnetZerotierNetworkId"),
	}

	h.RunScript(configureHostScript, conf, true, true)
}

type hostConfigData struct {
	*host
	ZerotierNetworkId  string
	PrivateNetwork     *net.IPNet
	PrivateNetworkMask net.IP
}

const configureHostScript = `
# This script is supposed to be idempotent
# Containers lose their connection to the bridge when running this script :-(

# echo commands and exit on error
set -v -e


if [ -n "{{.ZerotierNetworkId}}" ]
then
	zerotier-cli join {{.ZerotierNetworkId}}
fi

`

func (h *host) ConfigureNetwork() {
	conf := hostConfigData{
		host:               h,
		ZerotierNetworkId:  generalConfig("vnetZerotierNetworkId"),
		PrivateNetwork:     vnetNetwork(),
		PrivateNetworkMask: net.IP(vnetNetwork().Mask)}

	h.RunScript(configureHostNetworkScript, conf, true, true)
}

type hostConfigNetworkData struct {
	*host
	ZerotierNetworkId  string
	PrivateNetwork     *net.IPNet
	PrivateNetworkMask net.IP
}

const configureHostNetworkScript = `
# Containers lose their connection to the bridge when running this script :-(

# echo commands and exit on error
set -v -e

cat <<'EOF' > /etc/network/interfaces
# AUTOMATICALLY GENERATED - DO NOT EDIT

# This file describes the network interfaces available on your system
# and how to activate them. For more information, see interfaces(5).

source /etc/network/interfaces.d/*

# The loopback network interface
auto lo
iface lo inet loopback

# The primary network interface
auto eth0
allow-hotplug eth0
iface eth0 inet dhcp

auto br0
iface br0 inet static
    bridge_ports none
    address {{.PrivateIPv4}}
    netmask {{.PrivateNetworkMask}}
    up iptables -t nat -A POSTROUTING -s {{.PrivateNetwork}} -o eth0 -j MASQUERADE
    down iptables -t nat -D POSTROUTING -s {{.PrivateNetwork}} -o eth0 -j MASQUERADE

auto zt0
allow-hotplug zt0
iface zt0 inet manual
    up brctl addif br0 zt0
    down brctl delif br0 zt0
EOF

ifdown --all
ifup --all
`

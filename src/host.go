package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"infr/easyssh"
	"io"
	"net"
	"os"
	"strings"
)

type host struct {
	Name              string
	PublicIPv4        string
	PrivateIPv4       string
	SSHKnownHostsLine string
	sshClient         *ssh.Client
}

func findHost(name string) *host {
	for _, host := range config.Hosts {
		if host.Name == name {
			return host
		}
	}

	errorExit("Unknown host: %s", name)
	return nil
}

func allKnownHostLines() string {
	var out bytes.Buffer

	for _, h := range config.Hosts {
		fmt.Fprintln(&out, h.SSHKnownHostsLine)
	}

	return out.String()
}

func (h *host) FQDN() string {
	return h.Name + "." + infrDomain()
}

func (h *host) VnetFQDN() string {
	return h.Name + "." + vnetDomain()
}

func (h *host) AllLxcs() []*lxc {
	var lxcs []*lxc

	for i, l := range config.Lxcs {
		if l.Host == h.Name {
			lxcs = append(lxcs, config.Lxcs[i])
		}
	}

	return lxcs
}

func (h *host) SSHConfig() *easyssh.MakeConfig {
	return &easyssh.MakeConfig{
		User:   "manager",
		Server: h.PublicIPv4,
		Key:    "/.ssh/id_rsa",
		Port:   "22",
	}
}

func (h *host) RunCaptureStdout(cmd string, echo bool) string {
	ssh := h.SSHConfig()

	var stdout bytes.Buffer
	var stderr io.Writer

	if echo {
		stderr = os.Stderr
	}

	fmt.Printf("Capturing output on remote host: %s\n", h.Name)
	fmt.Println(cmd)
	err := ssh.RunCapture(cmd, &stdout, stderr)
	if err != nil {
		errorExit("Error running remote script: %s", err)
	}

	return stdout.String()
}

func (h *host) InstallSoftware() {
	h.SudoScript(files["host/install-software.sh"], nil)
	h.Upload(files["host/issue-ssl-certs"], nil, "/usr/local/bin/issue-ssl-certs", "www-data", "list", "0543")
}

func (h *host) Configure() {
	conf := hostConfigData{
		host:              h,
		ZerotierNetworkId: generalConfig("vnetZerotierNetworkId"),
		KnownHosts:        allKnownHostLines(),
	}

	h.SudoScript(files["host/configure.sh"], conf)
}

type hostConfigData struct {
	*host
	ZerotierNetworkId string
	KnownHosts        string
}

func (h *host) ConfigureNetwork() {
	conf := hostConfigNetworkData{
		host:               h,
		ZerotierNetworkId:  generalConfig("vnetZerotierNetworkId"),
		PrivateNetwork:     vnetNetwork(),
		PrivateNetworkMask: net.IP(vnetNetwork().Mask)}

	h.SudoScript(configureHostNetworkScript, conf)
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

func (h *host) retrieveHostSSHPubKey() {
	// SSH generates several host keys using different ciphers
	// We are retrieving the one that Debian 8 uses in 2016
	// This may not be robust
	pubkey := h.RunCaptureStdout("sudo cat /etc/ssh/ssh_host_ecdsa_key.pub", true)
	fields := strings.Fields(pubkey)
	h.SSHKnownHostsLine = fmt.Sprintf("%s,%s %s %s", h.FQDN(), h.PublicIPv4, fields[0], fields[1])
	saveConfig()
}

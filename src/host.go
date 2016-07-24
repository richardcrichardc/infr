package main

import (
	"bytes"
	"fmt"
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

func (h *host) RunScript(scriptTmpl string, data interface{}, echo, sudo bool) {

	script := executeTemplate(scriptTmpl, data)

	ssh := h.SSHConfig()

	fmt.Printf("Running script on remote host: %s\n", h.Name)
	err := ssh.RunScript(script, echo, sudo)
	if err != nil {
		errorExit("Error running remote script: %s", err)
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
	h.RunScript(installSoftwareScript, h, true, true)
}

const installSoftwareScript = `
# This script is idempotent

# echo commands and exit on error
set -v -e

# enable backports so we can install certbot

echo "deb http://ftp.debian.org/debian jessie-backports main" > /etc/apt/sources.list.d/backports.list
apt-get update

# install various packages
apt-get -y install lxc bridge-utils haproxy ssl-cert webfs btrfs-tools moreutils
apt-get -y install certbot -t jessie-backports

# create ssl directory for haproxy
mkdir -p /etc/haproxy/ssl

# create doc_root and .wellknown for certbot
mkdir -p /etc/haproxy/certbot/.well-known


# script for getting certbot to issue ssl certificates
cat << EOF > /etc/haproxy/issue-ssl-certs
#!/bin/bash

cat /etc/haproxy/https-domains | while read FQDN; do
  if [ "\$FQDN" != "" ]; then
  	certbot certonly --webroot --quiet --keep --agree-tos --webroot-path /etc/haproxy/certbot --email \$1 -d \$FQDN
  fi
done
EOF
chmod +x /etc/haproxy/issue-ssl-certs


# script for installing certs issued by certbot
cat << EOF > /etc/haproxy/install-ssl-certs
#!/bin/bash

# remove old certs and cert list
rm -f /etc/haproxy/ssl/*
truncate --size=0 /etc/haproxy/ssl-crt-list

# create default file used when HOST does not match any other certs
cat /etc/ssl/certs/ssl-cert-snakeoil.pem /etc/ssl/private/ssl-cert-snakeoil.key > /etc/haproxy/ssl/default.crt

cat /etc/haproxy/https-domains | while read FQDN; do
  if [ "\$FQDN" != "" ]; then
    LIVEDIR=/etc/letsencrypt/live/\$FQDN
    if [ -e "\$LIVEDIR" ]; then
        CERTFILE=/etc/haproxy/ssl/\$FQDN.crt
        echo \$CERTFILE >> /etc/haproxy/ssl-crt-list
        cat \$LIVEDIR/fullchain.pem \$LIVEDIR/privkey.pem  > \$CERTFILE
    fi
  fi
done
EOF
chmod +x /etc/haproxy/install-ssl-certs


# the certbot package has a cron job to renew certificates on a daily basis
# here we add a daily cron job to install the renewed certificates
ln -sf /etc/haproxy/install-ssl-certs /etc/cron.daily/install-ssl-certs

# config file for webfs - web server used for hosting .web-known directory used to issue ssl certificates
cat << EOF > /etc/webfsd.conf

web_root="/etc/haproxy/certbot"
web_ip="127.0.0.1"
web_port="9980"
web_user="www-data"
web_group="www-data"
web_extras="-j"
EOF

service webfs restart

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
		KnownHosts:        allKnownHostLines(),
	}

	h.RunScript(configureHostScript, conf, true, true)
}

type hostConfigData struct {
	*host
	ZerotierNetworkId string
	KnownHosts        string
}

const configureHostScript = `
# This script is supposed to be idempotent
# Containers lose their connection to the bridge when running this script :-(

# echo commands and exit on error
set -v -e


# Leave zerotier networks before joining so interface comes up as zt0
zerotier-cli listnetworks | cut -d ' ' -f 3 | while read networkId; do
   if [ "$networkId" != "<nwid>" ]; then
      zerotier-cli leave $networkId
   fi
done

# Join zerotier network
if [ -n "{{.ZerotierNetworkId}}" ]
then
	zerotier-cli join {{.ZerotierNetworkId}}
fi

cat <<'EOF' > /etc/ssh/ssh_known_hosts
{{.KnownHosts}}
EOF

# Configure HAProxy

cat <<'EOF' > /etc/haproxy/errors/no-backend.http
HTTP/1.0 404 Service Unavailable
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>404 Not Found</h1>
No such site.
</body></html>

EOF

cat <<'EOF' > /etc/haproxy/haproxy.cfg
{{.host.HAProxyCfg}}
EOF

cat <<'EOF' > /etc/haproxy/https-domains
{{.host.HAProxyHttpsDomains}}
EOF

/etc/haproxy/issue-ssl-certs richard@tawherotech.nz
/etc/haproxy/install-ssl-certs

service haproxy reload

`

func (h *host) ConfigureNetwork() {
	conf := hostConfigNetworkData{
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

func (h *host) retrieveHostSSHPubKey() {
	// SSH generates several host keys using different ciphers
	// We are retrieving the one that Debian 8 uses in 2016
	// This may not be robust
	pubkey := h.RunCaptureStdout("sudo cat /etc/ssh/ssh_host_ecdsa_key.pub", true)
	fields := strings.Fields(pubkey)
	h.SSHKnownHostsLine = fmt.Sprintf("%s,%s %s %s", h.Name+"."+needInfrDomain(), h.PublicIPv4, fields[0], fields[1])
	saveConfig()
}

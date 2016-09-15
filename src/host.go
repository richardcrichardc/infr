package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"strings"
)

type host struct {
	Name              string
	PublicIPv4        string
	PrivateIPv4       string
	SSHKnownHostsLine string
	sshClient         *ssh.Client
	down              bool
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

func (h *host) InstallSoftware() {
	h.Upload(infrDomain(), "/etc/infr-domain")
	h.UploadX(files("confedit"), "/usr/local/bin/confedit")
	h.UploadX(files("issue-ssl-certs"), "/usr/local/bin/issue-ssl-certs")
	h.UploadX(files("install-ssl-certs"), "/usr/local/bin/install-ssl-certs")
	h.UploadX(files("backup-host"), "/usr/local/bin/backup-host")
	h.UploadX(files("backup-all"), "/usr/local/bin/backup-all")
	h.UploadX(files("backup-send"), "/usr/local/bin/backup-send")
	h.UploadX(files("backups-to-cull"), "/usr/local/bin/backups-to-cull")
	h.Upload(files("infr-backup"), "/etc/cron.d/infr-backup")

	h.SudoScript(files("install-software.sh"), nil)

	// These must be run after packages have installed
	h.Upload(files("no-backend.http"), "/etc/haproxy/errors/no-backend.http")
	h.Upload(files("webfsd.conf"), "/etc/webfsd.conf")
	h.Sudo("service webfs restart")
	h.Upload(h.FQDN(), "/etc/mailname")
	h.Upload(needGeneralConfig("adminEmail"), "/etc/nullmailer/adminaddr")
	h.Upload(generalConfig("smtpRemote"), "/etc/nullmailer/remotes")
}

func (h *host) Configure() {
	h.Upload(allKnownHostLines(), "/etc/ssh/ssh_known_hosts")
	h.Upload(h.HAProxyCfg(), "/etc/haproxy/haproxy.cfg")
	h.Upload(strings.Join(h.HttpsTerminateDomains(), "\n")+"\n", "/etc/haproxy/https-domains")

	conf := map[string]string{
		"ZerotierNetworkId": generalConfig("vnetZerotierNetworkId"),
		"AdminEmail":        needGeneralConfig("adminEmail"),
	}

	h.SudoScript(files("configure.sh"), conf)
}

func (h *host) ConfigureNetwork() {
	conf := hostConfigNetworkData{
		host:               h,
		PrivateNetwork:     vnetNetwork(),
		PrivateNetworkMask: net.IP(vnetNetwork().Mask)}

	h.Upload(executeTemplate(files("interfaces"), conf), "/etc/network/interfaces")
	h.Sudo("ifdown --all && ifup --all")
}

type hostConfigNetworkData struct {
	*host
	PrivateNetwork     *net.IPNet
	PrivateNetworkMask net.IP
}

func (h *host) retrieveHostSSHPubKey() {
	// SSH generates several host keys using different ciphers
	// We are retrieving the one that Debian 8 uses in 2016
	// This may not be robust
	pubkey := h.SudoCaptureStdout("cat /etc/ssh/ssh_host_ecdsa_key.pub")
	fields := strings.Fields(pubkey)
	h.SSHKnownHostsLine = fmt.Sprintf("%s,%s %s %s", h.FQDN(), h.PublicIPv4, fields[0], fields[1])
	saveConfig()
}

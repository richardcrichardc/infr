package main

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"time"
)

var identityFile string

func (h *host) ConnectSSH() {
	var config ssh.ClientConfig

	config.User = "manager"

	key, err := parsePrivateKeyFile(identityFile)
	if err != nil {
		errorExit("Unable to parse SSH private key: %s", identityFile)
	}

	config.Auth = append(config.Auth, ssh.PublicKeys(key))

	config.Timeout = 5 * time.Second

	h.sshClient, err = ssh.Dial("tcp", h.FQDN()+":22", &config)
	if err != nil {
		errorExit("Unable to SSH to %s: %s", h.FQDN(), err)
	}
}

func parsePrivateKeyFile(keypath string) (ssh.Signer, error) {
	file := resolveHome(keypath)

	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (h *host) Remote(cmd string, stdinTmpl string, stdinData interface{}) {
	if h.sshClient == nil {
		h.ConnectSSH()
	}

	var stdin *bytes.Buffer

	if stdinTmpl != "" {
		stdin = executeTemplate(stdinTmpl, stdinData)
		logf("%s: %s <<EOF\n%s\nEOF", h.Name, cmd, stdin.String())
	} else {
		logf("%s: %s", h.Name, cmd)
	}

	session, err := h.sshClient.NewSession()
	if err != nil {
		logf("Unable to create session: %s", err)
	}
	defer session.Close()

	session.Stdout = log
	session.Stderr = log

	err = session.Run(cmd)
	if err != nil {
		logf("Remote command failed: %s", err)
	}
}

func (h *host) Sudo(cmd string) {
	h.Remote("sudo "+cmd, "", nil)
}

func (h *host) SudoScript(scriptTmpl string, data interface{}) {
	h.Remote("sudo bash", scriptTmpl, data)
}

func (h *host) Upload(fileTmpl string, data interface{}, path, owner, group, mask string) {
	h.Remote("sudo dd of="+path, fileTmpl, data)
	h.Sudo("chown " + owner + ":" + group + " " + path)
	h.Sudo("chmod " + mask + " " + path)
}

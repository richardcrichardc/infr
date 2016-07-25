package main

import (
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

func (h *host) Sudo(cmd string) {
	if h.sshClient == nil {
		h.ConnectSSH()
	}

	cmd = "sudo " + cmd

	logf("%s: %s", h.Name, cmd)

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

func (h *host) SudoScript(scriptTmpl string, data interface{}) {
	stdin := executeTemplate(scriptTmpl, data)

	if h.sshClient == nil {
		h.ConnectSSH()
	}

	logf("%s: Running script as root:\n%s\n", h.Name, stdin.String())

	session, err := h.sshClient.NewSession()
	if err != nil {
		logf("Unable to create session: %s", err)
	}
	defer session.Close()

	session.Stdin = stdin
	session.Stdout = log
	session.Stderr = log

	err = session.Run("sudo bash")
	if err != nil {
		logf("Remote script failed: %s", err)
	}
}

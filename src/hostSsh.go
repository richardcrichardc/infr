package main

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"time"
)

var identityFile string

func (h *host) ConnectSSH() error {
	var config ssh.ClientConfig

	config.User = "manager"

	key, err := parsePrivateKeyFile(identityFile)
	if err != nil {
		errorExit("Unable to parse SSH private key: %s", identityFile)
	}

	config.Auth = append(config.Auth, ssh.PublicKeys(key))

	config.Timeout = 5 * time.Second

	h.sshClient, err = ssh.Dial("tcp", h.FQDN()+":22", &config)
	return err
}

func (h *host) MustConnectSSH() {
	err := h.ConnectSSH()
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

func (h *host) Remote(cmd string, stdin string, stdout io.Writer) {
	logCmd := h.Name + ": " + cmd

	if stdout != nil {
		logCmd = logCmd + " > LOCAL"
	}

	if stdin != "" {
		logCmd = logCmd + " <<EOF\n" + stdin + "\nEOF"
	}

	logf("%s", logCmd)

	session, err := h.sshClient.NewSession()
	if err != nil {
		logf("Unable to create session: %s", err)
	}
	defer session.Close()

	if stdin != "" {
		session.Stdin = bytes.NewBufferString(stdin)
	}

	if stdout != nil {
		session.Stdout = stdout
	} else {
		session.Stdout = log
	}
	session.Stderr = log

	err = session.Run(cmd)
	if err != nil {
		errorExit("Remote command failed: %s", err)
	}
}

func (h *host) Sudo(cmd string) {
	h.Remote("sudo bash -c '"+cmd+"'", "", nil)
}

func (h *host) SudoScript(scriptTmpl string, data interface{}) {
	h.Remote("sudo bash", executeTemplate(scriptTmpl, data), nil)
}

func (h *host) Upload(file, path string) {
	h.Remote("sudo dd of="+path, file, nil)
}

func (h *host) UploadChownMod(file, path, owner, group, mask string) {
	h.Upload(file, path)
	h.Sudo("chown " + owner + ":" + group + " " + path)
	h.Sudo("chmod " + mask + " " + path)
}

func (h *host) UploadX(file, path string) {
	h.UploadChownMod(file, path, "root", "root", "0555")
}

func (h *host) SudoCaptureStdout(cmd string) string {
	var buf bytes.Buffer

	h.Remote("sudo bash -c '"+cmd+"'", "", &buf)

	return buf.String()
}

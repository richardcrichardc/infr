package main

import (
	"bufio"
	"bytes"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"strings"
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

	h.sshClient, err = ssh.Dial("tcp", h.PublicIPv4+":22", &config)

	if err == nil {
		h.down = false
	}

	return err
}

func (h *host) MustConnectSSH() {
	err := h.ConnectSSH()
	if err != nil {
		errorExit("Unable to SSH to %s: %s", h.FQDN(), err)
	}
	if !h.Lock() {
		errorExit("%s is locked. Wait for the existing operation to complete.", h.Name)
	}
}

func (h *host) Lock() bool {
	session, err := h.sshClient.NewSession()
	h.lockCheckErr(err)

	// Run script on remote host to do a flock and report back on success
	// The remote script then sleeps forever, and we leave the session open
	// So the lock is retained until we disconnect or exit

	// We may not have uploaded scripts to the remote host so we stream the script into Python
	// It gets a little complicated...

	// We need a TTY so the remote process will die when we disconnect
	err = session.RequestPty("xterm", 80, 40, ssh.TerminalModes{ssh.ECHO: 0})
	h.lockCheckErr(err)

	// Cannot pipe script directly into Python because python runs the REPL when directly
	// attached to a TTY
	cmd := "bash -c 'cat | python2'"

	// Need to add a CMD-D to designate EOF
	session.Stdin = bytes.NewBufferString(files("lock-host") + "\x04")

	// Collect all output to a buffered reader
	stdout, err := session.StdoutPipe()
	h.lockCheckErr(err)
	stderr, err := session.StderrPipe()
	h.lockCheckErr(err)
	out := bufio.NewReader(io.MultiReader(stdout, stderr))

	// Set it all in motion
	session.Start(cmd)

	// Read first line of output
	line, err := out.ReadString('\n')
	line = strings.TrimSpace(line)

	if line == "LOCKED" {
		return true
	} else if line == "ALREADY LOCKED" {
		return false
	}

	// Unexpected output log it all
	logf("Unexpected output when locking %s:\n%s\n", h.Name, line)
	out.WriteTo(log)
	errorExit("Unexpected behaviour when trying to lock host: %s", h.Name)
	return false
}

func (h *host) lockCheckErr(err error) {
	if err != nil {
		errorExit("Unable to lock host %s: %s", h.Name, err)
	}
}

func (h *host) Disconnect() {
	if h.sshClient != nil {
		h.sshClient.Close()
		h.sshClient = nil
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
		errorExit("Remote command failed")
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
		logf("Run failed: %s", err)
		errorExit("Remote command failed")
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

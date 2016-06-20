package evilbootstrap

import (
	"fmt"
	"infr/easyssh"
	"infr/util"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

func Install(currentAddress, currentRootPass, hostname, domainname, managerAuthKeys, lastPreseedURL string) (preseedURL string, err error) {
	defer msgLF()

	iPxeScript, preseedURL, err := genPxeScript(managerAuthKeys, lastPreseedURL)
	if err != nil {
		return preseedURL, err
	}

	if err := buildIPxe(iPxeScript); err != nil {
		return preseedURL, err
	}

	ssh := &easyssh.MakeConfig{
		User:     "root",
		Server:   currentAddress,
		Key:      "/.ssh/id_rsa",
		Password: currentRootPass,
		Port:     "22",
	}

	msg("Uploading ipxe.sub to root@%s...", currentAddress)
	if err := ssh.Scp("ipxe/src/bin/ipxe.usb"); err != nil {
		return preseedURL, err
	}

	msg("Starting reinstall...")
	if err := remote(ssh, "fsfreeze -f / && dd if=ipxe.usb of=/dev/vda bs=10M conv=fsync && reboot -f"); err != nil {
		// We expect remote machine to immediately reboot and ssh connection to hang
		_, isRunTimeout := err.(easyssh.RunTimeoutError)
		if !isRunTimeout {
			return preseedURL, err
		}
	}

	ssh = &easyssh.MakeConfig{
		User:   "manager",
		Server: currentAddress,
		Key:    "/.ssh/id_rsa",
		Port:   "22",
	}

	msg("Waiting for install to complete... (takes approx 10-15m, use virtual console to monitor)")
	var ready bool
	for !ready {
		_, err := ssh.Run("whoami")
		ready = err == nil
	}

	msg("Setting domainname...")
	// sudo does a DNS look up of hostname -
	// change hostname and hosts file at the same time so sudo doesn't fail to resolve the hostname
	hosts := fmt.Sprintf(hostsTemplate, hostname, domainname, hostname)
	if err := remote(ssh, "sudo sh -c 'hostnamectl set-hostname %s && echo \"%s\" >/etc/hosts'", hostname, hosts); err != nil {
		return preseedURL, err
	}

	return preseedURL, nil
}

func genPxeScript(managerAuthKeys, lastPreseedURL string) (script, preseedURL string, err error) {
	newGist := true

	// Add slash before newline to provide multiline string in preseed cfg
	escapedManagerAuthKeys := strings.Replace(managerAuthKeys, "\n", "\\\n", -1)

	preseedCfg := fmt.Sprintf(preseedTemplate, escapedManagerAuthKeys)

	if lastPreseedURL != "" {
		msg("Checking preseed.cfg at %s ...", lastPreseedURL)

		resp, err := http.Get(lastPreseedURL)

		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				lastPreseedBytes, _ := ioutil.ReadAll(resp.Body)
				if string(lastPreseedBytes) == preseedCfg {
					msg("Reusing Debian installer preseed.cfg Gist at: %s", lastPreseedURL)
					newGist = false
					preseedURL = lastPreseedURL
				}
			}
		}

	}

	if newGist {
		msg("Uploaded Debian installer preseed.cfg as new Gist...")

		preseedURL, err = CreateAnonymousGist(preseedCfg)
		if err != nil {
			return "", "", err
		}
		msg("Gist URL: %s", preseedURL)
	}

	return fmt.Sprintf(ipxeTemplate, preseedURL), preseedURL, nil
}

func buildIPxe(script string) error {

	// We are going to change directory, make sure we change back
	here, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func(whence string) {
		if err := os.Chdir(whence); err != nil {
			panic(err) // May not have chdired back to original directory
		}
	}(here)

	if util.Exists("ipxe") {
		msg("Assuming ./ipxe is clone of ipxe repo")
		msgLF()
	} else {
		if err := execToStdOutErr("git", "clone", "http://git.ipxe.org/ipxe.git"); err != nil {
			return err
		}
	}

	if err := os.Chdir("ipxe/src"); err != nil {
		return err
	}

	msg("Writing ipxe config files")

	if !util.Exists("config/local/evil") {
		if err := os.Mkdir("config/local/evil", 0777); err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(path.Join(".", "config/local/evil/general.h"), []byte("#define DOWNLOAD_PROTO_HTTPS\n"), 0700)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(".", "embedded-script"), []byte(script), 0700)
	if err != nil {
		return err
	}

	msg("Building ipxe")
	msgLF()
	if err := execToStdOutErr("make", "bin/ipxe.usb", "CONFIG=evil", "EMBED=embedded-script"); err != nil {
		return err
	}

	return nil
}

func execToStdOutErr(command string, arg ...string) error {
	cmd := exec.Command(command, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func remote(ssh *easyssh.MakeConfig, command string, args ...interface{}) error {
	command = fmt.Sprintf(command, args...)
	msg("Running remote command on %s@%s: %s", ssh.User, ssh.Server, command)

	output, err := ssh.Run(command)
	if err != nil {
		return err
	}
	msg("Output: %s", output)
	msgLF()

	return nil
}

var msgRunning bool
var msgLFDone chan bool

func msg(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)

	msgLF()
	msgRunning = true

	fmt.Print(time.Now().Format("15:04:05"), " ", text)

	go func() {
		for {
			ticker := time.NewTicker(time.Second)
			select {
			case <-ticker.C:
				fmt.Print("\r", time.Now().Format("15:04:05"), " ", text)
			case <-msgLFDone:
				return
			}
		}
	}()
}

func msgLF() {
	if msgRunning {
		msgLFDone <- true
		fmt.Println()
		msgRunning = false
	}
}

func init() {
	msgLFDone = make(chan bool)

}

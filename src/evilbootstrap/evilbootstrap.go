package evilbootstrap

import (
	"fmt"
	"infr/easyssh"
	"infr/util"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

func Install(currentAddress, currentRootPass, newFQDN, managerAuthKeys string) error {
	iPxeScript, err := genPxeScript(managerAuthKeys)
	if err != nil {
		return err
	}

	if err := buildIPxe(iPxeScript); err != nil {
		return err
	}

	ssh := &easyssh.MakeConfig{
		User:     "root",
		Server:   currentAddress,
		Key:      "/.ssh/id_rsa",
		Password: currentRootPass,
		Port:     "22",
	}

	log.Printf("Uploading ipxe.sub to root@%s...", currentAddress)
	if err := ssh.Scp("ipxe/src/bin/ipxe.usb"); err != nil {
		return err
	}

	if err := remote(ssh, "fsfreeze -f / && dd if=ipxe.usb of=/dev/vda bs=10M conv=fsync && reboot -f"); err != nil {
		// We expect remote machine to immediately reboot and ssh connection to hang
		_, isRunTimeout := err.(easyssh.RunTimeoutError)
		if !isRunTimeout {
			return err
		}
	}

	return nil
}

func genPxeScript(managerAuthKeys string) (string, error) {
	log.Printf("Uploaded debian install preseed.cfg")

	// Add slash before newline to provide multiline string in preseed cfg
	escapedManagerAuthKeys := strings.Replace(managerAuthKeys, "\n", "\\\n", -1)

	preseedCfg := fmt.Sprintf(preseedTemplate, escapedManagerAuthKeys)

	preeseedUrl, err := CreateAnonymousGist(preseedCfg)
	if err != nil {
		return "", err
	}
	log.Printf("preseed.cfg url = %s\n", preeseedUrl)

	return fmt.Sprintf(ipxeTemplate, preeseedUrl), nil
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
		log.Printf("Assuming ./ipxe is clone of ipxe repo")
	} else {
		if err := execToStdOutErr("git", "clone", "http://git.ipxe.org/ipxe.git"); err != nil {
			return err
		}
	}

	if err := os.Chdir("ipxe/src"); err != nil {
		return err
	}

	log.Printf("Writing ipxe config files")

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

	log.Printf("Building ipxe")
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

func remote(ssh *easyssh.MakeConfig, command string) error {
	log.Printf("Running remote command on %s@%s: %s", ssh.User, ssh.Server, command)

	output, err := ssh.Run(command)
	if err != nil {
		return err
	}
	log.Printf("Output: %s", output)

	return nil
}

package main

import (
	"fmt"
	"infr/util"
	"os"
	"path"
)

func initCmd(args []string) {
	resolveWorkDir()
	if util.Exists(path.Join(workDirPath, "config")) {
		errorExit("Working directory and configuration already exists: %s", workDirPath)
	}

	config.General = make(map[string]string)

	fmt.Println("Creating initial configuration. Run `infr help` or see README.md for more information...")

	config.General["vnetNetwork"] = util.PromptDefault("vnetNetwork", "10.8.0.0/16")
	config.General["vnetPoolStart"] = util.PromptDefault("vnetPoolStart", "10.8.1.0")
	config.General["dnsDomain"] = util.PromptNotBlank("dnsDomain")
	config.General["dnsPrefix"] = util.PromptDefault("dnsPrefix", "infr")
	config.General["vnetPrefix"] = util.PromptDefault("vnetPrefix", "vnet")
	keysAdd(resolveHome(util.PromptDefault("Initial ssh public keyfile", "$HOME/.ssh/id_rsa.pub")))
	config.General["adminEmail"] = util.PromptNotBlank("adminEmail")

	if err := os.MkdirAll(workDirPath, 0700); err != nil {
		errorExit("Unable to create working directory: %s", err)
	}

	saveConfig()
}

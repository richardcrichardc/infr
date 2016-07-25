package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

var config struct {
	General        map[string]string
	Keys           string
	Hosts          []*host
	Lxcs           []*lxc
	Backups        []*backup
	LastPreseedURL string
}

func loadConfig() {

	if err := os.MkdirAll(workDirPath, 0700); err != nil {
		errorExit("Error creating working directory: %s", err)
	}

	cdWorkDir()
	defer restoreCwd()

	jsonBytes, err := ioutil.ReadFile("config")
	if err != nil {
		if os.IsNotExist(err) {
			// No config
			config.General = make(map[string]string)
			return
		}
		errorExit("Error in loadConfig: %s", err)
	}

	err = json.Unmarshal(jsonBytes, &config)

	if err != nil {
		errorExit("Error in loadConfig: %s", err)
	}
}

func saveConfig() {
	cdWorkDir()
	defer restoreCwd()

	jsonBytes, err := json.MarshalIndent(config, "", "  ")

	if err != nil {
		errorExit("Error in saveConfig: %s", err)
	}

	err = ioutil.WriteFile("config", jsonBytes, 0600)

	if err != nil {
		errorExit("Error in saveConfig: %s", err)
	}
}

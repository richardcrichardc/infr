package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
)

var workDirPath string
var cwd string

var config struct {
	General        map[string]string
	Keys           string
	Hosts          []*host
	Lxcs           []*lxc
	Backups        []*backup
	LastPreseedURL string
}

func saveCwd() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

func restoreCwd() {
	err := os.Chdir(cwd)
	if err != nil {
		panic(err)
	}
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

func setupGlobalFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&workDirPath, "workdir", "$HOME/.infr", "Where configuration and other fluff is kept")
}

func cdWorkDir() {
	err := os.Chdir(workDirPath)
	if err != nil {
		errorExit("Unable to change to working directory '%s', use 'init' subcommand to make sure it exists.", workDirPath)
	}
}

func expandWorkDirPath() {
	workDirPath = resolveHome(workDirPath)
}

func resolveHome(path string) string {
	if strings.HasPrefix(path, "$HOME") {
		currentUser, _ := user.Current()
		if currentUser == nil || currentUser.HomeDir == "" {
			errorExit("Unable to resolve $HOME")
		}

		path = strings.Replace(path, "$HOME", currentUser.HomeDir, 1)
	}

	return path
}

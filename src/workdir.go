package main

import (
	"flag"
	"fmt"
	"infr/util"
	"os"
	"os/user"
	"strings"
)

var workDirPath string

func setupGlobalFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&workDirPath, "workdir", "$HOME/.infr", "Where configuration and other fluff is kept")
}

const initWorkDirHelp = `Usage: infr init

Create working directory and initial configuration files.
`

func initWorkDir(args []string) {

	if len(args) != 0 {
		errorHelpExit("init", "Too many arguments")
	}

	if err := os.MkdirAll(workDirPath, 0700); err != nil {
		errorExit("Error creating working directory: %s", err)
	}

	cdWorkDir()

	if util.Exists("config") {
		errorExit("Working director already initialised at: %s", workDirPath)
	}

	if err := os.Mkdir("config", 0700); err != nil {
		errorExit("Error creating config directory: %s", err)
	}

	fmt.Printf("Working dir and config initialised at: %s\n", workDirPath)
}

func cdWorkDir() {
	err := os.Chdir(workDirPath)
	if err != nil {
		errorExit("Unable to change to working directory '%s', use 'init' subcommand to make sure it exists. (%s)", workDirPath, err.Error())
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

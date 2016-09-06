package main

import (
	"os"
	"os/user"
	"strings"
)

var workDirPath string
var cwd string

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

func resolveWorkDir() {
	workDirPath = resolveHome(workDirPath)
}

func cdWorkDir() {
	err := os.Chdir(workDirPath)
	if err != nil {
		errorExit("Unable to change to working directory '%s', use 'init' subcommand to make sure it exists.", workDirPath)
	}
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

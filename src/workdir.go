package main

import (
	"flag"
	"fmt"
	"infr/util"
	"log"
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
		errorHelpAndExit("init", "Too many arguments")
	}

	if err := os.MkdirAll(workDirPath, 0700); err != nil {
		log.Fatal(err)
	}

	cdWorkDir()

	if util.Exists("config") {
		log.Fatalf("Working director already initialised at: %s", workDirPath)
	}

	if err := os.Mkdir("config", 0700); err != nil {
		log.Fatal(err)
	}

	log.Printf("Working dir and config initialised at: %s", workDirPath)
}

func cdWorkDir() {
	err := os.Chdir(workDirPath)
	if err != nil {
		log.Fatalf("Unable to change to working directory '%s', use 'init' subcommand to make sure it exists. (%s)", workDirPath, err.Error())
	}
}

func expandWorkDirPath() {
	workDirPath = resolveHome(workDirPath)
}

func resolveHome(path string) string {
	if strings.HasPrefix(path, "$HOME") {
		currentUser, _ := user.Current()
		if currentUser == nil || currentUser.HomeDir == "" {
			log.Fatalf("Unable to resolve $HOME")
		}

		path = strings.Replace(path, "$HOME", currentUser.HomeDir, 1)
	}

	return path
}

func configString(path string) string {
	segments := strings.Split(path, ".")

	for _, segment := range segments {
		fmt.Println(segment)
	}
	return ""
}

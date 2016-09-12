package main

import (
	//"infr/evilbootstrap"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

//convert help to html and roff (needs ruby-ronn)
//gog:generate ronn --roff help.ronn
//gog:generate mv help files/help
//go:generate scripts/compile-help

//inline files directory
//go:generate go run scripts/inliner.go

func main() {
	args := parseFlags(os.Args, setupGlobalFlags)
	if len(args) == 0 {
		errorExit("Please specify a command.")
	}
	cmd := args[0]

	saveCwd()

	// These commands do not require working directory or configuration
	switch cmd {
	case "init":
		initCmd(parseFlags(args, noFlags))
		os.Exit(0)
	case "help":
		helpCmd(parseFlags(args, noFlags))
		os.Exit(0)
	}

	// All other commands need a working directory and minimal configuration

	resolveWorkDir()
	openLog()
	loadConfig()

	switch cmd {
	case "config":
		configCmd(parseFlags(args, noFlags))
	case "keys":
		keysCmd(parseFlags(args, noFlags))
	case "hosts":
		hostsCmd(parseFlags(args, noFlags))
	case "host":
		hostCmd(parseFlags(args, noFlags))
	case "lxcs":
		lxcsCmd(parseFlags(args, noFlags))
	case "lxc":
		lxcCmd(parseFlags(args, noFlags))
	case "dns":
		dnsCmd(parseFlags(args, noFlags))
	case "backups":
		backupsCmd(parseFlags(args, noFlags))
	default:
		errorExit("Invalid command: %s", cmd)
	}
}

func parseFlags(args []string, setupFlags func(flagset *flag.FlagSet)) []string {
	flagset := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flagset.SetOutput(ioutil.Discard)
	setupFlags(flagset)
	err := flagset.Parse(args[1:])

	if err != nil {
		errorExit("%s: %s", args[0], err)
	}

	return flagset.Args()
}

func noFlags(flagset *flag.FlagSet) {
}

func setupGlobalFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&workDirPath, "w", "$HOME/.infr", "Workdir, where configuration and other fluff is kept")
	flagset.StringVar(&identityFile, "i", "$HOME/.ssh/id_rsa", "SSH identity file (private key)")
	flagset.BoolVar(&verbose, "v", false, "Output operation log to stdout")
	flagset.StringVar(&hostsDown, "D", "", "Comma separated list of hosts that are down")
}

func errorExit(format string, formatArgs ...interface{}) {
	fmt.Fprintf(os.Stderr, format, formatArgs...)
	fmt.Fprintf(os.Stderr, "\n\nRun '%s help' to view documentation. Remote command log can be found in %s/log. \n", os.Args[0], workDirPath)
	os.Exit(1)
}

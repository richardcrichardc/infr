package main

import (
	//"infr/evilbootstrap"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

//go:generate go run scripts/inliner.go

func main() {
	args := parseFlags(os.Args, setupGlobalFlags)
	expandWorkDirPath()
	saveCwd()
	openLog()
	loadConfig()

	if len(args) == 0 {
		errorExit("Please specify a command.")
	}

	switch args[0] {
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
	case "help":
		helpCmd(parseFlags(args, noFlags))
	default:
		errorExit("Invalid command: %s", args[0])
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
}

func errorExit(format string, formatArgs ...interface{}) {
	fmt.Fprintf(os.Stderr, format, formatArgs...)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Run '%s help' to view documentation.\n", os.Args[0])
	os.Exit(1)
}

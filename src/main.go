package main

import (
	//"infr/evilbootstrap"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	args := parseFlags(os.Args, setupGlobalFlags)
	expandWorkDirPath()

	saveCwd()
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

func errorExit(format string, formatArgs ...interface{}) {
	fmt.Fprintf(os.Stderr, format, formatArgs...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

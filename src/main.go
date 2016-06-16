package main

import (
	//"infr/evilbootstrap"
	"flag"
	"fmt"
	"os"
)

type subcommand struct {
	name       string
	setupFlags func(flagset *flag.FlagSet)
	run        func(args []string)
	desc       string
	help       string
}

var cmds []subcommand

func main() {

	cmds = []subcommand{
		{"init", nil, initWorkDir, "Initialise working directory and configuration file", initWorkDirHelp},
		{"keys", keyFlags, keys, "List, add and remove ssh keys for managing hosts and containers", keysHelp},
		{"hosts", hostsFlags, hosts, "List, add and remove hosts", hostsHelp},
	}

	if len(os.Args) == 1 {
		usageAndExit()
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	if cmdName == "help" {
		if len(cmdArgs) != 1 {
			usageAndExit()
		}

		help(cmdArgs[0])
	} else {
		cmd := lookupCmd(cmdName)

		if cmd == nil {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmdName)
			usageAndExit()
		}

		flagset := setupFlags(cmd)

		if err := flagset.Parse(cmdArgs); err != nil {
			errorHelpExit(cmd.name, "Error parsing flags: %s\n\n", err)
		}

		expandWorkDirPath()

		cmd.run(flagset.Args())
	}
}

func lookupCmd(name string) *subcommand {
	for _, cmd := range cmds {
		if name == cmd.name {
			return &cmd
		}
	}

	return nil
}

func setupFlags(cmd *subcommand) *flag.FlagSet {
	flagset := flag.NewFlagSet(cmd.name, flag.ContinueOnError)
	if cmd.setupFlags != nil {
		cmd.setupFlags(flagset)
	}

	setupGlobalFlags(flagset)

	return flagset
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, `Infr is ....

Usage:

	infr command [arguments]

The commands are:
`)

	for _, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "    %-10s %s\n", cmd.name, cmd.desc)
	}

	fmt.Fprintln(os.Stderr, `
Use "infr help [command]" for more information about a command.
`)

	os.Exit(1)
}

func help(topic string) {
	cmd := lookupCmd(topic)

	if cmd != nil && cmd.help != "" {

		flagset := setupFlags(cmd)

		fmt.Fprintln(os.Stderr, cmd.help)
		fmt.Fprintln(os.Stderr, "Flags:")
		flagset.PrintDefaults()
		fmt.Fprintln(os.Stderr)

		return
	}

	fmt.Fprintln(os.Stderr, "Sorry no help for: ", topic)
	os.Exit(1)
}

func errorHelpExit(topic, format string, formatArgs ...interface{}) {
	fmt.Fprintf(os.Stderr, format, formatArgs...)
	fmt.Fprint(os.Stderr, "\n\n")
	help(topic)
	os.Exit(1)
}

func errorExit(format string, formatArgs ...interface{}) {
	fmt.Fprintf(os.Stderr, format, formatArgs...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

package main

import (
	//"infr/evilbootstrap"
	"flag"
	"fmt"
	"os"
)

type command struct {
	name    string
	desc    string
	help    string
	subCmds []subCommand
}

type subCommand struct {
	name       string
	setupFlags func(flagset *flag.FlagSet)
	run        func(args []string)
	help       string
}

var cmds []command

func main() {

	cmds = []command{
		{"init", "Initialise working directory and configuration file", initWorkDirHelp,
			[]subCommand{{"", nil, initWorkDir, ""}}},
		{"config", "Set and view configuration used by other commands", configHelp,
			[]subCommand{
				{"view", nil, configViewCmd, ""},
				{"set", nil, configSetCmd, ""},
				{"unset", nil, configUnsetCmd, ""}}},
		{"keys", "List, add and remove ssh keys for managing hosts and containers", keysHelp,
			[]subCommand{
				{"list", nil, keysListCmd, ""},
				{"add", nil, keysAddCmd, ""},
				{"remove", nil, keysRemoveCmd, ""}}},
		{"hosts", "List, add and remove hosts that containers run on", hostsHelp,
			[]subCommand{
				{"list", nil, hostsListCmd, hostsListHelp},
				{"add", hostsAddFlags, hostsAddCmd, hostsAddHelp},
				{"remove", nil, hostsRemoveCmd, hostsRemoveHelp}}},
		{"dns", "Manage DNS", "",
			[]subCommand{{"", nil, dnsListCmd, ""}}},
		{"help", "", "",
			[]subCommand{{"", nil, helpCmd, ""}}},
	}

	if len(os.Args) == 1 {
		usageAndExit()
	}

	cmd, subCmd, cmdArgs := lookupCmd(os.Args[1:])

	if cmd == nil {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		usageAndExit()
	}

	if subCmd == nil {
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", os.Args[2])
		usageAndExit()
	}

	flagset := setupFlags(subCmd)

	if err := flagset.Parse(cmdArgs); err != nil {
		errorHelpExit(cmd.name, "Error parsing flags: %s\n\n", err)
		usageAndExit()
	}

	expandWorkDirPath()

	subCmd.run(flagset.Args())
	//}
}

func lookupCmd(args []string) (*command, *subCommand, []string) {
	cmdName := args[0]
	for _, cmd := range cmds {
		if cmdName == cmd.name {
			/* Match first sub command if there is:
			* Only one subcommand
			* No args after command
			* First arg after command is an option
			 */
			if len(cmd.subCmds) == 1 || len(args) == 1 || args[1][0:1] == "-" {
				return &cmd, &cmd.subCmds[0], args[1:]
			}

			subCmdName := args[1]
			for _, subCmd := range cmd.subCmds {
				if subCmdName == subCmd.name {
					return &cmd, &subCmd, args[2:]
				}
			}
			return &cmd, nil, nil
		}
	}

	return nil, nil, nil
}

func setupFlags(subCmd *subCommand) *flag.FlagSet {
	flagset := flag.NewFlagSet(subCmd.name, flag.ContinueOnError)
	if subCmd.setupFlags != nil {
		subCmd.setupFlags(flagset)
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
		if cmd.desc != "" {
			fmt.Fprintf(os.Stderr, "    %-10s %s\n", cmd.name, cmd.desc)
		}
	}

	fmt.Fprintln(os.Stderr, `
Use "infr help [command]" for more information about a command.
`)

	os.Exit(1)
}

func help(topic string) {
	helpCmd([]string{topic})
}

func helpCmd(args []string) {
	if len(args) != 1 {
		usageAndExit()
	}

	cmd, _, _ := lookupCmd(args)

	if cmd != nil && cmd.help != "" {

		simpleHelp := true
		for _, subCmd := range cmd.subCmds {
			if subCmd.help != "" {
				simpleHelp = false
				break
			}
		}

		if simpleHelp {
			fmt.Fprintln(os.Stderr, cmd.help)
			fmt.Fprintln(os.Stderr, "Flags:")
			flagset := setupFlags(&cmd.subCmds[0])
			flagset.PrintDefaults()
			fmt.Fprintln(os.Stderr)
		} else {

			fmt.Fprintln(os.Stderr, cmd.help)
			fmt.Fprintln(os.Stderr)

			for _, subCmd := range cmd.subCmds {
				if subCmd.help != "" {
					fmt.Fprintf(os.Stderr, "SUBCOMMAND: %s", subCmd.help)
					if subCmd.setupFlags != nil {
						fmt.Fprintln(os.Stderr, "Flags:")
						flagset := flag.NewFlagSet(subCmd.name, flag.ContinueOnError)
						subCmd.setupFlags(flagset)
						flagset.PrintDefaults()
					}
					fmt.Fprintln(os.Stderr)
					fmt.Fprintln(os.Stderr)
				}
			}
			fmt.Fprintln(os.Stderr, "COMMON FLAGS:")
			fmt.Fprintln(os.Stderr)
			flagset := flag.NewFlagSet("", flag.ContinueOnError)
			setupGlobalFlags(flagset)
			flagset.PrintDefaults()
			fmt.Fprintln(os.Stderr)

		}
		return
	}

	fmt.Fprintln(os.Stderr, "Sorry no help for: ", args[0])
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

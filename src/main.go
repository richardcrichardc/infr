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
			errorHelpAndExit(cmd.name, "Error parsing flags: %s\n\n", err)
		}

		expandWorkDirPath()

		cmd.run(cmdArgs)
	}

	/*
		app := cli.NewApp()

		app.Flags = []cli.Flag{
			cli.StringFlag{
				Name:  "workdir",
				Value: "$HOME/.infr",
				Usage: "Working directory where state is kept",
			},
		}

		app.Commands = []cli.Command{
			{
				Name:      "init",
				Usage:     "Initialise working directory",
				ArgsUsage: "[keyfile]",
				Action:    createConfig,
			},
			{
				Name:      "addkeys",
				Usage:     "Add SSH key which can be used to manage containers",
				ArgsUsage: "[keyfile]",
				Before:    loadConfig,
				Action:    addKeys,
			},
		}

		app.Run(os.Args)
	*/

	/*
	   	currentAddress := "45.63.27.225"
	   	currentRootPass := ""
	   	newFQDN := "kaiiwi@tawherotech.nz"
	   	managerAuthKeys := `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDKteJko6gySsW+z1zMD2xsSwhl5x12P3SVC5RFLO6COIaVjFeC3ELUEmEZtZM81n6bUMXvmXoXRE3UZWNTmWQlbX4P4tdAVvfgp+HnQ7a/qttpj7PxheLxMNaOUczFcF+GIqNKJ9x4vcXH+v3Lzt16ZB1PZSrzyOWExZ03iU5+hAa9QBgEndSLTePjEBX9zgGawyH/H44/LeRzZ+Rhov1A96ufinT73jtv3lq5MSsovkRLMq7BQB22yVllEkeRTzaqAuVc4W6lwa/NK0LHWteBQy1PZUv8L5zPPTNsgm6HgdNRDi7blamraTJR/2QEqPAWey56eECLDk4z8Rzh7Si/ richardc@kensalrise
	   ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDcOvlenEefsuKBfV89WTCIktQZC2692k+vD2emTXKSAiU+QKgpBhK3TzKeid+zQWZBlmK1wBSd+JS/kLPtB6KpLwLNDf/RgtTLwxqiG58eBIxQ/TGaWQMNRLVUEivDJnqvudeKvM1YiiRbFpBXj33KLewS6bDo5EdFLIzExh6OxaDP5qli54PxEUjAq6R8OZWPqCFE3F9SLjyQIB5iBqI6bwKrs3q6Di8sUt4RkuUGLAS4ev5KvgMaNdSJRq1ulGmauD1gzHRCYlmrSwbk6MCFB4FyL8tCqoWUs3HkAByWFS9jk324pE/hzMXkkFStSu3H6T8qYblpmpUrb4cCT63P donald@silver`

	   	err := evilbootstrap.Install(currentAddress, currentRootPass, newFQDN, managerAuthKeys)
	   	if err != nil {
	   		fmt.Println(err)
	   	}

	*/
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

func errorHelpAndExit(topic, format string, formatArgs ...interface{}) {
	fmt.Fprintf(os.Stderr, format, formatArgs...)
	fmt.Fprint(os.Stderr, "\n\n")
	help(topic)
	os.Exit(1)
}

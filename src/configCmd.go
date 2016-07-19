package main

import (
	"fmt"
)

func configCmd(args []string) {
	if len(args) == 0 {
		configViewCmd(args)
	} else {
		switch args[0] {
		case "view":
			configViewCmd(parseFlags(args, noFlags))
		case "set":
			configSetCmd(parseFlags(args, noFlags))
		case "unset":
			configUnsetCmd(parseFlags(args, noFlags))
		default:
			errorExit("Invalid command: %s", args[0])
		}
	}
}

func configViewCmd(args []string) {
	switch len(args) {
	case 0:

		for key, value := range config.General {
			fmt.Printf("%s: %s\n", key, value)
		}

	case 1:
		name := args[0]
		value, ok := config.General[name]

		if ok {
			fmt.Println(value)
		} else {
			errorExit("No config for: %s", name)
		}
	default:
		errorExit("Too many arguments for 'view'.")
	}
}

func configSetCmd(args []string) {
	if len(args) == 0 || len(args) > 2 {
		errorExit("Wrong number of arguments for 'set'.")
	}

	config.General[args[0]] = args[1]
	saveConfig()
}

func configUnsetCmd(args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'unset'.")
	}

	delete(config.General, args[0])
	saveConfig()
}

func generalConfig(name string) string {
	return config.General[name]
}

func needGeneralConfig(name string) string {
	value, ok := config.General[name]
	if !ok {
		errorExit("'%s' not configured. Use `infr config set %s <value>` to configure.", name, name)
	}

	return value
}

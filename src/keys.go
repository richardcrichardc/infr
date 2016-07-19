package main

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

func keysCmd(args []string) {
	if len(args) == 0 {
		keysListCmd(args)
	} else {
		switch args[0] {
		case "list":
			keysListCmd(parseFlags(args, noFlags))
		case "add":
			keysAddCmd(parseFlags(args, noFlags))
		case "remove":
			keysRemoveCmd(parseFlags(args, noFlags))
		default:
			errorExit("Invalid command: %s", args[0])
		}
	}
}

const keysHelp = `Usage: infr keys [list|add|remove] [keyfile]

List, add, or remove ssh keys used for managing hosts and containers.

The contents of [keyfile] should be in the format of .ssh/authorized_keys.
`

var keysAddFile, keysRemoveFile string

func keysListCmd(args []string) {
	if len(args) != 0 {
		errorExit("Too many arguments for 'list'.")
	}

	fmt.Println(config.Keys)
}

func keysAddCmd(args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'add'.")
	}

	newKeysBytes, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Println("Error reading keyfile: %s", err.Error())
	}
	newKeys := strings.Split(string(newKeysBytes), "\n")

	confKeys := strings.Split(config.Keys, "\n")

	allKeys := stripEmptyStrings(uniqueStrings(append(confKeys, newKeys...)))
	config.Keys = strings.Join(allKeys, "\n")

	saveConfig()
}

func keysRemoveCmd(args []string) {
	if len(args) != 1 {
		errorExit("Wrong number of arguments for 'remove'.")
	}

	removeKeysBytes, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Println("Error reading keyfile: %s", err.Error())
	}
	removeKeys := strings.Split(string(removeKeysBytes), "\n")

	confKeys := strings.Split(config.Keys, "\n")

	allKeys := removeStrings(confKeys, removeKeys)
	config.Keys = strings.Join(allKeys, "\n")

	saveConfig()
}

func uniqueStrings(s []string) []string {

	if len(s) < 2 {
		return s
	}

	var sorted, unique []string

	sorted = append(sorted, s...)
	sort.Strings(sorted)

	prev := sorted[0]
	for _, cur := range sorted[1:] {
		if prev != cur {
			unique = append(unique, prev)
			prev = cur
		}
	}
	unique = append(unique, prev)

	return unique
}

func stripEmptyStrings(s []string) []string {
	var out []string

	for _, cur := range s {
		if cur != "" {
			out = append(out, cur)
		}
	}

	return out
}

func removeStrings(s, remove []string) []string {
	var out []string

sloop:
	for _, cur := range s {
		for _, r := range remove {
			if cur == r {
				continue sloop
			}
		}
		out = append(out, cur)
	}
	return out
}

func needKeys() string {
	if config.Keys == "" {
		errorExit("No ssh keys configured. Use `infr keys add <keyfile>` to add keys.")
	}
	return config.Keys
}

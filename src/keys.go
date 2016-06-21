package main

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

const keysHelp = `Usage: infr keys [list|add|remove] [keyfile]

List, add, or remove ssh keys used for managing hosts and containers.

The contents of [keyfile] should be in the format of .ssh/authorized_keys.
`

var keysAddFile, keysRemoveFile string

func keysListCmd(args []string) {
	if len(args) != 0 {
		errorHelpExit("keys", "Too many arguments for 'list'.")
	}

	var keys string
	loadConfig("keys", &keys)
	fmt.Println(keys)
}

func keysAddCmd(args []string) {
	if len(args) != 1 {
		errorHelpExit("keys", "Wrong number of arguments for 'add'.")
	}

	newKeysBytes, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Println("Error reading keyfile: %s", err.Error())
	}
	newKeys := strings.Split(string(newKeysBytes), "\n")

	var confKeysStr string
	loadConfig("keys", &confKeysStr)
	confKeys := strings.Split(confKeysStr, "\n")

	allKeys := stripEmptyStrings(uniqueStrings(append(confKeys, newKeys...)))
	allKeysStr := strings.Join(allKeys, "\n")

	saveConfig("keys", allKeysStr)
}

func keysRemoveCmd(args []string) {
	if len(args) != 1 {
		errorHelpExit("keys", "Wrong number of arguments for 'remove'.")
	}

	removeKeysBytes, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Println("Error reading keyfile: %s", err.Error())
	}
	removeKeys := strings.Split(string(removeKeysBytes), "\n")

	var confKeysStr string
	loadConfig("keys", &confKeysStr)
	confKeys := strings.Split(confKeysStr, "\n")

	allKeys := removeStrings(confKeys, removeKeys)
	allKeysStr := strings.Join(allKeys, "\n")

	saveConfig("keys", allKeysStr)
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

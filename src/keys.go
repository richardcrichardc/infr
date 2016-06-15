package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

const keysHelp = `Usage: infr keys [-a keyfile] [-d keyfile]

Add, delete and list ssh keys used for managing hosts and containers.

The new set of public keys will be listed after adding and deleting keys specified by the
-a and -d options. Run without options to list current keys.

`

var keysAddFile, keysRemoveFile string

func keyFlags(fs *flag.FlagSet) {
	fs.StringVar(&keysAddFile, "a", "", "Add keys in specified file")
	fs.StringVar(&keysRemoveFile, "r", "", "Remove keys in specified file")
}

func keys(args []string) {
	var keysAdd, keysRemove []string

	if keysAddFile != "" {
		keysAddBytes, err := ioutil.ReadFile(keysAddFile)
		if err != nil {
			fmt.Println("Error reading keyfile: %s", err.Error())
		}
		keysAdd = strings.Split(string(keysAddBytes), "\n")
	}

	if keysRemoveFile != "" {
		keysRemoveBytes, err := ioutil.ReadFile(keysRemoveFile)
		if err != nil {
			fmt.Println("Error reading keyfile: %s", err.Error())
		}
		keysRemove = strings.Split(string(keysRemoveBytes), "\n")
	}

	var confKeysStr string
	loadConfig("keys", &confKeysStr)

	confKeys := strings.Split(confKeysStr, "\n")

	allKeys := stripEmptyStrings(
		uniqueStrings(
			removeStrings(
				append(confKeys, keysAdd...), keysRemove)))

	allKeysStr := strings.Join(allKeys, "\n")

	saveConfig("keys", allKeysStr)
	fmt.Println(allKeysStr)
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

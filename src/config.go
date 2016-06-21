package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	ospath "path"
	"path/filepath"
	"strings"
)

const configHelp = `Usage: infr config view [<name>]
       infr config set <name> <value>
       infr config unset <name>

View, set, or unset configuration strings used by other commands.

`

func configViewCmd(args []string) {
	switch len(args) {
	case 0:
		cdConfigDir()
		err := filepath.Walk("general", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				var value string
				loadConfig(path, &value)
				fmt.Printf("%s: %s\n", path[8:], value)
			}

			return nil
		})

		if err != nil {
			errorExit("Error traversing config values: %s", err)
		}

	case 1:
		name := args[0]
		var value string
		success := loadConfig("general/"+name, &value)

		if success {
			fmt.Println(value)
		} else {
			errorExit("No config for: %s", name)
		}
	default:
		errorHelpExit("config", "Too many arguments for 'view'.")
	}
}

func configSetCmd(args []string) {
	if len(args) == 0 || len(args) > 2 {
		errorHelpExit("config", "Wrong number of arguments for 'set'.")
	}

	name := args[0]
	value := args[1]

	saveConfig("general/"+name, value)
}

func configUnsetCmd(args []string) {
	if len(args) != 1 {
		errorHelpExit("config", "Wrong number of arguments for 'unset'.")
	}

	removeConfig("general/" + args[0])
}

func cdConfigDir() {
	cdWorkDir()
	err := os.Chdir("config")
	if err != nil {
		errorExit("Unable to change to config directory: %s", err.Error())
	}
}

func loadConfig(path string, value interface{}) bool {
	checkPath(path)

	cdConfigDir()

	jsonBytes, err := ioutil.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		errorExit("Error in loadConfig(%s): %s", path, err)
	}

	err = json.Unmarshal(jsonBytes, value)

	if err != nil {
		errorExit("Error in loadConfig(%s): %s", path, err)
	}

	return true
}

func saveConfig(path string, value interface{}) {
	checkPath(path)

	jsonBytes, err := json.Marshal(value)

	if err != nil {
		errorExit("Error in saveConfig(%s): %s", path, err)
	}

	cdConfigDir()

	dir := ospath.Dir(path)
	if dir != "." {
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			errorExit("Error in saveConfig(%s): %s", path, err)
		}
	}

	err = ioutil.WriteFile(path, jsonBytes, 0600)

	if err != nil {
		errorExit("Error in saveConfig(%s): %s", path, err)
	}
}

func removeConfig(path string) {
	checkPath(path)
	cdConfigDir()
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		errorExit("Error in removeConfig(%s): %s", path, err)
	}

	// Delete any empty parent directories
	for err == nil && path != "." {
		path = ospath.Dir(path)
		err = os.Remove(path)
	}

}

func checkPath(path string) {
	if strings.Contains(path, "..") {
		errorExit("Aborting due to dangerous '..' in config name: %s", path)
	}
}

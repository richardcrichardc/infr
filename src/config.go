package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	ospath "path"
)

func cdConfigDir() {
	cdWorkDir()
	err := os.Chdir("config")
	if err != nil {
		errorExit("Unable to change to config directory: %s\n", err.Error())
	}
}

func loadConfig(path string, value interface{}) bool {
	cdConfigDir()

	jsonBytes, err := ioutil.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		errorExit("Error in loadConfig(%s): %s\n", path, err)
	}

	err = json.Unmarshal(jsonBytes, value)

	if err != nil {
		errorExit("Error in loadConfig(%s): %s\n", path, err)
	}

	return true
}

func saveConfig(path string, value interface{}) {

	jsonBytes, err := json.Marshal(value)

	if err != nil {
		errorExit("Error in saveConfig(%s): %s\n", path, err)
	}

	cdConfigDir()

	dir := ospath.Dir(path)
	if dir != "." {
		os.MkdirAll(dir, 0700)
	}

	err = ioutil.WriteFile(path, jsonBytes, 0600)

	if err != nil {
		errorExit("Error in saveConfig(%s): %s\n", path, err)
	}
}

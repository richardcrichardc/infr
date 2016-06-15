package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	ospath "path"
)

func cdConfigDir() {
	cdWorkDir()
	err := os.Chdir("config")
	if err != nil {
		fmt.Printf("Unable to change to config directory: %s\n", err.Error())
		os.Exit(1)
	}
}

func loadConfig(path string, value interface{}) bool {
	cdConfigDir()

	jsonBytes, err := ioutil.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		fmt.Printf("Error in loadConfig(%s): %s\n", path, err)
		os.Exit(1)
	}

	err = json.Unmarshal(jsonBytes, value)

	if err != nil {
		fmt.Printf("Error in loadConfig(%s): %s\n", path, err)
		os.Exit(1)
	}

	return true
}

func saveConfig(path string, value interface{}) {

	jsonBytes, err := json.Marshal(value)

	if err != nil {
		fmt.Printf("Error in saveConfig(%s): %s\n", path, err)
		os.Exit(1)
	}

	cdConfigDir()

	dir := ospath.Dir(path)
	if dir != "." {
		os.MkdirAll(dir, 0700)
	}

	err = ioutil.WriteFile(path, jsonBytes, 0600)

	if err != nil {
		fmt.Printf("Error in saveConfig(%s): %s\n", path, err)
		os.Exit(1)
	}
}

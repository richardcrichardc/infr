package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	ospath "path"
)

func cdConfigDir() {
	cdWorkDir()
	err := os.Chdir("config")
	if err != nil {
		log.Fatalf("Unable to change to config directory: ", err.Error())
	}
}

func loadConfig(path string, value interface{}) bool {
	cdConfigDir()

	jsonBytes, err := ioutil.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		log.Fatalf("Error in loadConfig(%s): %s", path, err)
	}

	err = json.Unmarshal(jsonBytes, value)

	if err != nil {
		log.Fatalf("Error in loadConfig(%s): %s", path, err)
	}

	return true
}

func saveConfig(path string, value interface{}) {

	jsonBytes, err := json.Marshal(value)

	if err != nil {
		log.Fatalf("Error in saveConfig(%s): %s", path, err)
	}

	cdConfigDir()

	dir := ospath.Dir(path)
	if dir != "." {
		os.MkdirAll(dir, 0700)
	}

	err = ioutil.WriteFile(path, jsonBytes, 0600)

	if err != nil {
		log.Fatalf("Error in saveConfig(%s): %s", path, err)
	}
}

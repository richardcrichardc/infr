package main

import (
	"io/ioutil"
	"io"
	"log"
	"encoding/json"
)


func loadConfig(path string, value, defaultValue, interface{}) {
	cdWorkDir()

	jsonBytes, err := ioutil.ReadFile(path)

	if err != nil {
		if os.IsNotExists(err) {
			value := defaultValue
			return
		}

		log.Fatalf("Error in loadConfig(%s): %s", path, err)
	}

	err = json.Unmarshal(jsonBytes, value)

	if err != nil {
		log.Fatalf("Error in loadConfig(%s): %s", path, err)
	}
}


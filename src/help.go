package main

import (
	"bytes"
	"os"
	"os/exec"
)

func helpCmd(args []string) {
	if len(args) > 0 {
		errorExit("Too many arguments for 'infr help'")
	}

	cmd := exec.Command("man", "-l", "-")
	cmd.Stdin = bytes.NewBufferString(files("help"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		errorExit("Error formatting help: %s", err)
	}
}

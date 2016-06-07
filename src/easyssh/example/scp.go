package main

import (
	"fmt"

	"gopkg.in/hypersleep/easyssh.v0"
)

func main() {
	// Create MakeConfig instance with remote username, server address and path to private key.
	ssh := &easyssh.MakeConfig{
		User:   "john",
		Server: "example.com",
		Key:    "/.ssh/id_rsa",
		Port: "22",
	}

	// Call Scp method with file you want to upload to remote server.
	err := ssh.Scp("/home/core/zipkin.rb")

	// Handle errors
	if err != nil {
		panic("Can't run remote command: " + err.Error())
	} else {
		fmt.Println("success")

		response, _ := ssh.Run("ls -al zipkin.rb")

		fmt.Println(response)
	}
}

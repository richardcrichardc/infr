package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Prompt(msg string) string {
	fmt.Printf("%s: ", msg)
	reader := bufio.NewReader(os.Stdin)
	str, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from stdin: %s", err)
		os.Exit(1)
	}
	return strings.TrimSpace(str)
}

func PromptDefault(msg, defaultValue string) string {
	newMsg := fmt.Sprintf("%s (default=%s)", msg, defaultValue)
	result := Prompt(newMsg)

	if result == "" {
		return defaultValue
	}

	return result
}

func PromptNotBlank(msg string) string {
	var result string

	for result == "" {
		result = Prompt(msg)
	}

	return result
}

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err) // We cannot handle any other errors
	}
	return true
}

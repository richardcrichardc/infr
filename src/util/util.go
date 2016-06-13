package util

import (
	"bufio"
	"fmt"
	"os"
)

func Prompt(msg string) (string, error) {
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
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

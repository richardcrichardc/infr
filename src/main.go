package main

import (
	"fmt"
	"infr/evilbootstrap"
	"os"
)

func main() {
	fmt.Printf("hello, world: %v \n", os.Args)

	var addr, rootPassword string
	switch len(os.Args) {
	case 2:
		addr = os.Args[1]
	case 3:
		addr = os.Args[1]
		rootPassword = os.Args[1]
	default:
		fmt.Printf("Usage: %s addr [rootPassword]\n", os.Args[0])
		return
	}

	err := evilbootstrap.Install(addr, rootPassword)

	if err != nil {
		fmt.Println(err)
	}
}

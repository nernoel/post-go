package main

import (
	"fmt"
	"os"

	"postgo/commands"
)

func main() {
	if err := commands.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

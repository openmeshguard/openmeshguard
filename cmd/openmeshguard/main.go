package main

import (
	"fmt"
	"os"
)

func main() {
	if err := newRootCommand(defaultVersionInfo()).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

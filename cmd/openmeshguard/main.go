package main

import (
	"fmt"
	"os"
)

func main() {
	ignoreBrokenPipeSignal()
	if err := newRootCommand(defaultVersionInfo()).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

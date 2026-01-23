package main

import (
	"os"

	"github.com/omriariav/workspace-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

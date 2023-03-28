package main

import (
	"os"

	"github.com/deta/pc-cli/cmd"
)

func main() {
	cmd := cmd.NewSpaceCmd()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

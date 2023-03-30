package main

import (
	"os"

	"github.com/deta/space/cmd"
)

func main() {
	cmd := cmd.NewSpaceCmd()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

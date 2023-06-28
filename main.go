package main

import (
	"os"

	"github.com/deta/space/cmd"
)

func main() {
	if err := cmd.NewSpaceCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

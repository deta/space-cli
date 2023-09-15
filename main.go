package main

import (
	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"os"

	"github.com/deta/space/cmd"
)

func main() {
	if err := cmd.NewSpaceCmd().Execute(); err != nil {
		utils.StdErrLogger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		os.Exit(1)
	}
}

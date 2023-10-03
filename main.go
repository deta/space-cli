package main

import (
	"errors"
	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/pkg/components"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"os"

	"github.com/deta/space/cmd"
)

func main() {
	if err := cmd.NewSpaceCmd().Execute(); err != nil {
		// user prompt cancellation is not an error
		if errors.Is(err, components.ErrPromptCancelled) {
			return
		}
		utils.StdErrLogger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		os.Exit(1)
	}
}

package cmd

import (
	"fmt"

	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newCmdOpen() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open",
		Short:   "open current project in browser",
		PreRunE: CheckAll(CheckExists("dir"), CheckNotEmpty("id")),
		RunE:    open,
	}

	cmd.Flags().StringP("id", "i", "", "project id of project to open")
	cmd.Flags().StringP("dir", "d", "./", "src of project to open")

	return cmd
}

func open(cmd *cobra.Command, args []string) error {
	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	projectDir, _ := cmd.Flags().GetString("dir")
	projectID, _ := cmd.Flags().GetString("id")

	if !cmd.Flags().Changed("id") {
		var err error
		projectID, err = runtime.GetProjectID(projectDir)
		if err != nil {
			return fmt.Errorf("%s Failed to get project id %w", emoji.ErrorExclamation, err)
		}
	}

	logger.Printf("Opening project in default browser...\n")
	if err := browser.OpenURL(fmt.Sprintf("%s/%s", builderUrl, projectID)); err != nil {
		return fmt.Errorf("%s Failed to open browser window %w", emoji.ErrorExclamation, err)
	}

	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}
	return nil
}

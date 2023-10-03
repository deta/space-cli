package cmd

import (
	"fmt"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/runtime"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newCmdOpen() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "open",
		Short:    "Open your local project in the Builder UI",
		PreRunE:  utils.CheckAll(utils.CheckExists("dir"), utils.CheckNotEmpty("id")),
		PostRunE: utils.CheckLatestVersion,

		RunE: open,
	}

	cmd.Flags().StringP("id", "i", "", "project id of project to open")
	cmd.Flags().StringP("dir", "d", "./", "src of project to open")

	return cmd
}

func open(cmd *cobra.Command, args []string) error {

	projectDir, _ := cmd.Flags().GetString("dir")
	projectID, _ := cmd.Flags().GetString("id")

	if !cmd.Flags().Changed("id") {
		var err error
		projectID, err = runtime.GetProjectID(projectDir)
		if err != nil {
			return fmt.Errorf("failed to get the project id, %w", err)
		}
	}

	utils.Logger.Printf("Opening project in default browser...\n")
	if err := browser.OpenURL(fmt.Sprintf("%s/%s", utils.BuilderUrl, projectID)); err != nil {
		return fmt.Errorf("failed to open a browser window, %w", err)
	}

	return nil
}

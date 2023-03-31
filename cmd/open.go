package cmd

import (
	"fmt"
	"os"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newCmdOpen() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open",
		Short:   "Open your local project in the Builder UI",
		PreRunE: shared.CheckAll(shared.CheckExists("dir"), shared.CheckNotEmpty("id")),
		Run:     open,
	}

	cmd.Flags().StringP("id", "i", "", "project id of project to open")
	cmd.Flags().StringP("dir", "d", "./", "src of project to open")

	return cmd
}

func open(cmd *cobra.Command, args []string) {

	projectDir, _ := cmd.Flags().GetString("dir")
	projectID, _ := cmd.Flags().GetString("id")

	if !cmd.Flags().Changed("id") {
		var err error
		projectID, err = runtime.GetProjectID(projectDir)
		if err != nil {
			shared.Logger.Printf("%s Failed to get project id: %s", emoji.ErrorExclamation, err)
			os.Exit(1)
		}
	}

	shared.Logger.Printf("Opening project in default browser...\n")
	if err := browser.OpenURL(fmt.Sprintf("%s/%s", shared.BuilderUrl, projectID)); err != nil {
		shared.Logger.Printf("%s Failed to open browser window %s", emoji.ErrorExclamation, err)
		os.Exit(1)
	}

}

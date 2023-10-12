package cmd

import (
	"os"
	"path/filepath"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/emoji"

	"github.com/spf13/cobra"
)

func newCmdPull() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "pull",
		Short:    "Pull the latest release of your project from Space",
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")
			if !cmd.Flags().Changed("id") {
				var err error
				projectID, err = runtime.GetProjectID(projectDir)
				if err != nil {
					utils.Logger.Printf("%s Failed to get project id: %s", emoji.ErrorExclamation, err)
					return err
				}
			}

			resp, err := utils.Client.GetProjectZipball(&api.GetProjectZipballRequest{ID: projectID})
			if err != nil {
				return err
			}

			zipFilePath := filepath.Join(projectDir, resp.Name)
			err = os.WriteFile(zipFilePath, resp.Data, 0644)
			if err != nil {
				utils.Logger.Printf("%s Failed to write zip file: %s", emoji.ErrorExclamation, err)
				return err
			}

			utils.Logger.Printf("%s Project pulled successfully to %s", emoji.Check, zipFilePath)
			return nil
		},
	}

	cmd.Flags().StringP("id", "i", "", "`project_id` of project")
	cmd.Flags().StringP("dir", "d", "./", "src of project")

	cmd.MarkFlagDirname("dir")

	return cmd
}

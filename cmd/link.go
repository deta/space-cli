package cmd

import (
	"errors"
	"fmt"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/auth"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/deta/space/pkg/components/text"
	"github.com/spf13/cobra"
)

func newCmdLink() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "link [flags]",
		Short:    "Link a local directory with an existing project",
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")

			if !cmd.Flags().Changed("id") {
				utils.Logger.Printf("Grab the %s of the project you want to link to using Teletype.\n\n", styles.Code("Project ID"))

				if projectID, err = selectLinkProjectID(); err != nil {
					return err
				}
			}

			if err := link(projectDir, projectID); err != nil {
				return err
			}

			return nil
		},
		PreRunE: utils.CheckAll(
			utils.CheckExists("dir"),
			utils.CheckNotEmpty("id"),
		),
	}

	cmd.Flags().StringP("id", "i", "", "project id of project to link")
	cmd.Flags().StringP("dir", "d", "./", "src of project to link")

	return cmd
}

func selectLinkProjectID() (string, error) {
	promptInput := text.Input{
		Prompt:      "Project ID",
		Placeholder: "",
		Validator: func(value string) error {
			if value == "" {
				return fmt.Errorf("please provide a valid id, empty project id is not valid")
			}
			return nil
		},
	}

	return text.Run(&promptInput)
}

func link(projectDir string, projectID string) error {
	projectRes, err := utils.Client.GetProject(&api.GetProjectRequest{ID: projectID})
	if err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			utils.Logger.Println(utils.LoginInfo())
			return err
		}
		if errors.Is(err, api.ErrProjectNotFound) {
			utils.Logger.Println(styles.Errorf("%s No project found. Please provide a valid Project ID.", emoji.ErrorExclamation))
			return err
		}

		utils.Logger.Println(styles.Errorf("%s Failed to link project, %s", emoji.ErrorExclamation, err.Error()))
		return err
	}

	err = runtime.StoreProjectMeta(projectDir, &runtime.ProjectMeta{ID: projectRes.ID, Name: projectRes.Name, Alias: projectRes.Alias})
	if err != nil {
		utils.Logger.Printf("failed to link project: %s", err)
		return err
	}

	utils.Logger.Println(styles.Greenf("%s Project", emoji.Link), styles.Pink(projectRes.Name), styles.Green("was linked!"))
	utils.Logger.Println(utils.ProjectNotes(projectRes.Name, projectRes.ID))
	return nil
}

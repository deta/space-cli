package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/deta/space/cmd/shared"
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
		Use:   "link [flags]",
		Short: "link code to project",
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")

			if !cmd.Flags().Changed("id") {
				shared.Logger.Printf("Grab the %s of the project you want to link to using Teletype.\n\n", styles.Code("Project ID"))

				if projectID, err = selectLinkProjectID(); err != nil {
					os.Exit(1)
				}
			}

			if err := link(projectDir, projectID); err != nil {
				os.Exit(1)
			}
		},
		PreRunE: shared.CheckAll(
			shared.CheckExists("dir"),
			shared.CheckNotEmpty("id"),
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
	if err := runtime.AddSpaceToGitignore(projectDir); err != nil {
		shared.Logger.Println("failed to add .space to .gitignore, %w", err)
		return err
	}

	projectRes, err := shared.Client.GetProject(&api.GetProjectRequest{ID: projectID})
	if err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			shared.Logger.Println(shared.LoginInfo())
			return err
		}
		if errors.Is(err, api.ErrProjectNotFound) {
			shared.Logger.Println(styles.Errorf("%s No project found. Please provide a valid Project ID.", emoji.ErrorExclamation))
			return err
		}

		shared.Logger.Println(styles.Errorf("%s Failed to link project, %s", emoji.ErrorExclamation, err.Error()))
		return err
	}

	err = runtime.StoreProjectMeta(projectDir, &runtime.ProjectMeta{ID: projectRes.ID, Name: projectRes.Name, Alias: projectRes.Alias})
	if err != nil {
		shared.Logger.Printf("failed to link project: %s", err)
		return err
	}

	shared.Logger.Println(styles.Greenf("%s Project", emoji.Link), styles.Pink(projectRes.Name), styles.Green("was linked!"))
	shared.Logger.Println(shared.ProjectNotes(projectRes.Name, projectRes.ID))
	return nil
}

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/spf13/cobra"
)

func newCmdLink() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link [flags]",
		Short: "link code to project",
		Run:   link,
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

func link(cmd *cobra.Command, args []string) {
	var err error

	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	projectDir, _ := cmd.Flags().GetString("dir")

	var projectID string
	if cmd.Flags().Changed("id") {
		projectID, _ = cmd.Flags().GetString("id")
	} else {
		logger.Printf("Grab the %s of the project you want to link to using Teletype.\n\n", styles.Code("Project ID"))

		if projectID, err = selectLinkProjectID(); err != nil {
			os.Exit(1)
		}
	}

	if err := runtime.AddSpaceToGitignore(projectDir); err != nil {
		logger.Println("failed to add .space to .gitignore, %w", err)
		os.Exit(1)
	}

	projectRes, err := client.GetProject(&api.GetProjectRequest{ID: projectID})
	if err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			logger.Println(LoginInfo())
			os.Exit(1)
		}
		if errors.Is(err, api.ErrProjectNotFound) {
			logger.Println(styles.Errorf("%s No project found. Please provide a valid Project ID.", emoji.ErrorExclamation))
			os.Exit(1)
		}

		logger.Println(styles.Errorf("%s Failed to link project, %s", emoji.ErrorExclamation, err.Error()))
		os.Exit(1)
	}

	err = runtime.StoreProjectMeta(projectDir, &runtime.ProjectMeta{ID: projectRes.ID, Name: projectRes.Name, Alias: projectRes.Alias})
	if err != nil {
		logger.Printf("failed to link project: %s", err)
		os.Exit(1)
	}

	logger.Println(styles.Greenf("%s Project", emoji.Link), styles.Pink(projectRes.Name), styles.Green("was linked!"))

	logger.Println(projectNotes(projectRes.Name, projectRes.ID))
	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}
}

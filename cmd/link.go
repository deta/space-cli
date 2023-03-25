package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/spf13/cobra"
)

var (
	linkProjectID  string
	linkProjectDir string
	linkCmd        = &cobra.Command{
		Use:   "link [flags]",
		Short: "link code to project",
		RunE:  link,
	}
)

func init() {
	linkCmd.Flags().StringVarP(&linkProjectID, "id", "i", "", "project id of project to link")
	linkCmd.Flags().StringVarP(&linkProjectDir, "dir", "d", "./", "src of project to link")
	rootCmd.AddCommand(linkCmd)
}

func selectLinkProjectID() (string, error) {
	promptInput := text.Input{
		Prompt:      "Project ID",
		Placeholder: "",
		Validator:   projectIDValidator,
	}

	return text.Run(&promptInput)
}

var (
	NoProjectFoundMsg = styles.Errorf("%s No project found. Please provide a valid Project ID.", emoji.ErrorExclamation)
)

func link(cmd *cobra.Command, args []string) error {
	logger.Println()

	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	var err error

	linkProjectDir = filepath.Clean(linkProjectDir)

	runtimeManager := runtime.NewManager(linkProjectDir)
	isProjectInitialized, err := runtimeManager.IsProjectInitialized()
	if err != nil {
		return err
	}

	if isProjectInitialized {
		existingProjectMeta, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return err
		}
		logger.Printf("%s This directory is already linked to a project named \"%s\".\n", emoji.Cowboy, existingProjectMeta.Name)
		logger.Println(projectNotes(existingProjectMeta.Name, existingProjectMeta.ID))
		cm := <-c
		if cm.err == nil && cm.isLower {
			logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
		}
		return nil
	}

	if isFlagEmpty(linkProjectID) {
		logger.Printf("You can link your local src code to a Space project!\n\n")
		logger.Printf("Grab the %s of the project you want to link to using Teletype.\n\n", styles.Code("Project ID"))
		linkProjectID, err = selectLinkProjectID()
		if err != nil {
			return err
		}
	}

	err = runtime.AddSpaceToGitignore(projectDir)
	if err != nil {
		return fmt.Errorf("failed to add .space to .gitignore, %w", err)
	}

	projectRes, err := client.GetProject(&api.GetProjectRequest{ID: linkProjectID})
	if err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			logger.Println(LoginInfo())
			return nil
		}
		if errors.Is(err, api.ErrProjectNotFound) {
			logger.Println(NoProjectFoundMsg)
			return nil
		}
		return err
	}

	err = runtime.StoreProjectMeta(projectDir, &runtime.ProjectMeta{ID: projectRes.ID, Name: projectRes.Name, Alias: projectRes.Alias})
	if err != nil {
		return fmt.Errorf("failed to link project, %w", err)
	}

	logger.Println(styles.Greenf("%s Project", emoji.Link), styles.Pink(projectRes.Name), styles.Green("was linked!"))

	logger.Println(projectNotes(projectRes.Name, projectRes.ID))
	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}

	return nil
}

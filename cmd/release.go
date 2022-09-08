package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/spf13/cobra"
)

var (
	releaseDir       string
	revisionID       string
	releaseProjectID string
	releaseVersion   string
	releaseDesc      string

	releaseCmd = &cobra.Command{
		Use:   "release [flags]",
		Short: "create release for a project",
		RunE:  release,
	}
)

func init() {
	releaseCmd.Flags().StringVarP(&releaseDir, "dir", "d", "./", "src of project to release")
	releaseCmd.Flags().StringVarP(&releaseProjectID, "id", "i", "", "project id of an existing project")
	releaseCmd.Flags().StringVarP(&revisionID, "rid", "r", "", "revision id for release")
	releaseCmd.Flags().StringVarP(&releaseVersion, "version", "v", "", "version for the release")
	releaseCmd.Flags().StringVarP(&releaseDesc, "short-desc", "s", "", "ashort description for the release")
	rootCmd.AddCommand(releaseCmd)
}

func selectRevisionID() (string, error) {
	promptInput := text.Input{
		Prompt:      "What is the revision id?",
		Placeholder: "",
		Validator:   emptyPromptValidator,
	}

	return text.Run(&promptInput)
}

func selectProjectID() (string, error) {
	promptInput := text.Input{
		Prompt:      "What is your project id?",
		Placeholder: "",
		Validator:   emptyPromptValidator,
	}

	return text.Run(&promptInput)
}

func selectVersion() (string, error) {
	promptInput := text.Input{
		Prompt:      "What is the version for the release?",
		Placeholder: "",
	}

	return text.Run(&promptInput)
}

func selectDescription() (string, error) {
	promptInput := text.Input{
		Prompt:      "What is a short description for your release?",
		Placeholder: "",
	}

	return text.Run(&promptInput)
}

func release(cmd *cobra.Command, args []string) error {

	releaseDir = filepath.Clean(releaseDir)

	runtimeManager, err := runtime.NewManager(&releaseDir, true)
	if err != nil {
		return err
	}

	isProjectInitialized, err := runtimeManager.IsProjectInitialized()
	if err != nil {
		return err
	}

	if isFlagEmpty(revisionID) {
		revisionID, err = selectRevisionID()
		if err != nil {
			return fmt.Errorf("problem while trying to get revision id from prompt, %w", err)
		}
	}

	if isProjectInitialized {
		projectMeta, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return err
		}
		releaseProjectID = projectMeta.ID
	} else if isFlagEmpty(releaseProjectID) {
		logger.Printf("No project initialized.\n\n")

		releaseProjectID, err = selectProjectID()
		if err != nil {
			return fmt.Errorf("problem while trying to get project id to release from text prompt, %w", err)
		}
	}

	if isFlagEmpty(releaseVersion) {
		releaseVersion, err = selectVersion()
		if err != nil {
			return fmt.Errorf("problem while trying to get version from prompt, %w", err)
		}
	}

	if isFlagEmpty(releaseDesc) {
		releaseDesc, err = selectDescription()
		if err != nil {
			return fmt.Errorf("problem while trying to get short description from prompt, %w", err)
		}
	}

	// TODO: start promotion
	// TODO: promotion logs
	logger.Println("Creating a release...")
	logs := make(chan string)
	go func() {
		err = client.GetReleaseLogs(&api.GetReleaseLogsRequest{ID: "project-id"}, logs)
		if err != nil {
			logger.Fatal(err)
		}
		close(logs)
	}()

	for msg := range logs {
		logger.Print(msg)
	}
	return nil
}

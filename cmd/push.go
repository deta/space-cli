package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/spf13/cobra"
)

var (
	pushProjectId  string
	pushProjectDir string
	pushCmd        = &cobra.Command{
		Use:   "push [flags]",
		Short: "push code",
		RunE:  push,
	}
)

func init() {
	pushCmd.Flags().StringVarP(&pushProjectId, "id", "i", "", "what's your project id?")
	pushCmd.Flags().StringVarP(&pushProjectDir, "dir", "d", "./", "where's your project that you want to push?")
	rootCmd.AddCommand(pushCmd)
}

func projectIdValidator(projectId string) error {
	if projectId == "" {
		return fmt.Errorf("please provide a valid id, empty project id is not valid")
	}
	return nil
}

func selectPushProjectId() (string, error) {
	promptInput := text.Input{
		Prompt:      "What's the project id?",
		Placeholder: "",
		Validator:   projectIdValidator,
	}

	return text.Run(&promptInput)
}

func push(cmd *cobra.Command, args []string) error {

	var err error

	pushProjectDir = filepath.Clean(pushProjectDir)

	runtimeManager, err := runtime.NewManager(&pushProjectDir, false)
	if err != nil {
		return err
	}

	isProjectInitialized, err := runtimeManager.IsProjectInitialized()
	if err != nil {
		return err
	}

	if !isProjectInitialized {
		if isFlagEmpty(pushProjectId) {

			logger.Printf("No project initialized.\n\n")
			pushProjectId, err = selectPushProjectId()
			if err != nil {
				return fmt.Errorf("problem while trying to get project id to push from text prompt, %w", err)
			}
		}

		logger.Println("Pushing code....")
		logger.Println("TODO: push code request with project id")
		return nil
	}

	projectMeta, err := runtimeManager.GetProjectMeta()
	if err != nil {
		return err
	}
	pushProjectId = projectMeta.ID

	logger.Println("Pushing code...")
	logger.Println("TODO: push code request with project id...")
	return nil
}

package cmd

import (
	"os"
	"path/filepath"

	"github.com/deta/pc-cli/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	projectId string
	pushCmd   = &cobra.Command{
		Use:   "push [flags]",
		Short: "push code",
		RunE:  push,
	}
)

func init() {
	pushCmd.Flags().StringVarP(&projectId, "project-id", "p", "", "what's your project id?")
	rootCmd.AddCommand(pushCmd)
}

func push(cmd *cobra.Command, args []string) error {

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	dir = filepath.Base(wd)

	runtimeManager, err := runtime.NewManager(&dir, false)
	if err != nil {
		return err
	}

	isProjectInitialized, err := runtimeManager.IsProjectInitialized()
	if err != nil {
		return err
	}

	if !isProjectInitialized {
		if isFlagEmpty(projectId) {
			logger.Println("No project initialized. Provide a project id via args or create a new project before running deta push.")
			return nil
		}
		logger.Println("Pushing code....")
		logger.Println("TODO: push code request with project id")
		return nil
	}

	projectMeta, err := runtimeManager.GetProjectMeta()
	if err != nil {
		return err
	}
	projectId = projectMeta.ID

	logger.Println("Pushing code...")
	logger.Println("TODO: push code request with project id...")
	return nil
}

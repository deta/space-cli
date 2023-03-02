package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var (
	openProjectID  string
	openProjectDir string
	openCmd        = &cobra.Command{
		Use:   "open",
		Short: "open current project in browser",
		RunE:  open,
	}
)

func init() {
	openCmd.Flags().StringVarP(&openProjectID, "id", "i", "", "project id of project to open")
	openCmd.Flags().StringVarP(&openProjectDir, "dir", "d", "./", "src of project to open")
	rootCmd.AddCommand(openCmd)
}

func open(cmd *cobra.Command, args []string) error {
	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	var err error

	openProjectDir = filepath.Clean(openProjectDir)

	runtimeManager, err := runtime.NewManager(&openProjectDir, false)
	if err != nil {
		return err
	}

	isProjectInitialized, err := runtimeManager.IsProjectInitialized()
	if err != nil {
		return err
	}

	var projectName string

	// check if project is initialized
	if isProjectInitialized {
		projectMeta, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return err
		}
		openProjectID = projectMeta.ID
		projectName = projectMeta.Name
	} else if isFlagEmpty(openProjectID) {
		logger.Printf("No project was found in the current directory.\n\n")
		logger.Printf("To create a new project run %s", styles.Code("space new"))

		return nil
	}

	logger.Printf("Opening project %s in default browser...\n", styles.Pink(projectName))

	var url = fmt.Sprintf("%s/%s", builderUrl, openProjectID)
	err = browser.OpenURL(url)

	if err != nil {
		return fmt.Errorf("%s Failed to open browser window %w", emoji.ErrorExclamation, err)
	}

	return nil
}

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/choose"
	"github.com/deta/pc-cli/pkg/components/confirm"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/deta/pc-cli/pkg/util/fs"
	"github.com/deta/pc-cli/shared"
	"github.com/spf13/cobra"
)

var (
	projectName string
	projectDir  string

	newCmd = &cobra.Command{
		Use:   "new [flags]",
		Short: "new project",
		RunE:  new,
	}

	templates = []string{"todo", "notes", "hello-world", "blank"}
)

const (
	// blank template
	Blank string = "blank"
)

func init() {
	newCmd.Flags().StringVarP(&projectName, "name", "n", "", "what's your project name?")
	newCmd.Flags().StringVarP(&projectDir, "dir", "d", "./", "where is this project?")
	rootCmd.AddCommand(newCmd)
}

func logScannedMicros(micros []*shared.Micro) {
	logger.Println("Scanned micros:")
	for _, micro := range micros {
		microMsg := fmt.Sprintf("L %s\n", micro.Name)
		microMsg += fmt.Sprintf("\tL src: %s\n", micro.Src)
		microMsg += fmt.Sprintf("\tL engine: %s\n\n", micro.Engine)
		logger.Printf(microMsg)
	}
}

func projectNameValidator(projectName string) error {

	if len(projectName) < 4 {
		return fmt.Errorf("project name %s must be at least 4 characters long", projectName)
	}

	if len(projectName) > 16 {
		return fmt.Errorf("project name %s must be at most 16 characters long", projectName)
	}

	return nil
}

func selectProjectName() (string, error) {
	promptInput := text.Input{
		Prompt:      "What's your project's name?",
		Placeholder: "default",
		Validator:   projectNameValidator,
	}

	return text.Run(&promptInput)
}

func selectProjectDir() (string, error) {
	promptInput := text.Input{
		Prompt:      "Where do you want to create your project?",
		Placeholder: "./",
	}

	return text.Run(&promptInput)
}

func selectTemplate() (string, error) {
	promptInput := choose.Input{
		Prompt:  "What type of micro do you want to create?",
		Choices: templates,
	}

	m, err := choose.Run(&promptInput)
	return templates[m.Cursor], err

}

func confirmCreateProjectWithDetectedConfig() (bool, error) {
	return confirm.Run(&confirm.Input{Prompt: "Do you want to create a project with the auto-detected configuration?"})
}

func new(cmd *cobra.Command, args []string) error {
	var err error

	if isFlagEmpty(projectName) {
		projectName, err = selectProjectName()
		if err != nil {
			return fmt.Errorf("problem while trying to get project's name through text prompt, %v", err)
		}
	}

	projectDir = filepath.Clean(projectDir)

	runtimeManager, err := runtime.NewManager(&projectDir, false)
	if err != nil {
		return err
	}

	isProjectInitialized, err := runtimeManager.IsProjectInitialized()
	if err != nil {
		return err
	}

	if isProjectInitialized {
		logger.Println("A project already exists in this dir.")
		return nil
	}

	isEmpty, err := fs.IsEmpty(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to check contents of dir %s, %v", projectDir, err)
	}

	// prompt to start from template
	if isEmpty {
		_, err := selectTemplate()
		if err != nil {
			return fmt.Errorf("problem while trying to get template from select prompt, %v", err)
		}

		logger.Println("Downloading template files...")
		logger.Printf("Creating project %s...", projectName)
		logger.Println("TODO: create project request")
		return nil
		// TODO: download template files
		// TODO: make api call to create project
		// TODO: write project id from create project api request's response to .space/meta
		// TODO: improve successfull create project logs
	}

	isManifestPresent, err := manifest.IsManifestPresent(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to check for manifest file in dir %s, %v", projectDir, err)
	}

	// yes yaml
	if isManifestPresent {
		logger.Printf(`Creating project "%s" with the existing manifest...`, projectName)
		logger.Println("TODO: create project request")
		// TODO: parse manifest and validate
		// TODO: make api call to create project
		// TODO: write project id from create project api request's response to .space/meta
		// TODO: improve successfull create project logs
		return nil
	}

	// no yaml present

	// auto-detect micros
	autoDetectedMicros, err := scanner.Scan(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to auto detect runtimes/frameworks for project %s, %v", projectName, err)
	}

	if len(autoDetectedMicros) > 0 {
		// prompt user for confirmation to create project with detected configuration
		logScannedMicros(autoDetectedMicros)
		create, err := confirmCreateProjectWithDetectedConfig()
		if err != nil {
			return fmt.Errorf("problem while trying to get confirmation to create project with the auto-detected configuration from confirm prompt, %v", err)
		}

		// create project with detected config
		if create {
			logger.Printf("Creating project %s with detected config....\n", projectName)
			logger.Println("TODO: create project request")
			return nil
		}
	}

	// don't create project with detected config, create blank project, point to docs
	logger.Printf("Creating blank project %s...\n", projectName)
	logger.Println("read docs...")
	logger.Println("TODO: create project request")
	return nil
}

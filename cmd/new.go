package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/pkg/confirm"
	"github.com/deta/pc-cli/pkg/text"
	"github.com/spf13/cobra"
)

var (
	name                 string
	dir                  string
	confirmCreateProject bool

	newCmd = &cobra.Command{
		Use:   "new [flags]",
		Short: "Create a new project",
		RunE:  new,
	}
)

func init() {
	newCmd.Flags().StringVarP(&name, "name", "n", "", "name of the new project")
	newCmd.Flags().StringVarP(&dir, "dir", "d", "", "where the project is created")
	newCmd.Flags().BoolVarP(&confirmCreateProject, "confirm", "c", false, "prefill missing arguments")

	rootCmd.AddCommand(newCmd)
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

func getDefaultAlias(projectName string) string {
	aliasRegexp := regexp.MustCompile(`/([^\w])/g,`)
	return aliasRegexp.ReplaceAllString(projectName, "")
}

func selectProjectName() (name string, err error) {
	promptInput := text.Input{
		Prompt:      "What's your project's name?",
		Placeholder: "default",
		Validator:   projectNameValidator,
	}

	return text.Run(&promptInput)
}

func selectDir() (dir string, err error) {
	promptInput := text.Input{
		Prompt:      "Where do you want to create your project?",
		Placeholder: "./",
	}

	return text.Run(&promptInput)
}

func confirmCreateMicro() (bool, error) {
	return confirm.Run(&confirm.Input{Prompt: "Do you want to create a micro now"})
}

func isFlagSet(flag string) bool {
	return flag != ""
}

func createBlankProjectSuccessLogs(projectName string, projectId string) string {
	message := `
Project "%s" created successfully! https://deta.space/builder/%s
We created a "deta.yml" for you, it contains the configuration for the project & tells Deta how to deploy and run it.

To add a micro run "deta new micro" or modify your "deta.yml" manually.

To deploy this project, follow the instructions in the docs: https://docs.deta.space/deploy`

	return fmt.Sprintf(message, projectName, projectId)
}

func createBlankProject() {
	// check if manifest exists
	manifestExists, err := manifest.ManifestExists(dir)
	if err != nil {
		logger.Printf("Failed to check if a manifest file already exists, %v\n", err)
		return
	}
	if manifestExists {
		logger.Printf(`"deta.yml" already exists in the dir %s`, dir)
		return
	}

	// create blank project
	logger.Printf("Creating new blank project \"%s\"...\n", name)
	// TODO: make create project request
	successLogs := createBlankProjectSuccessLogs(name, name)
	logger.Println(successLogs)
}

func createProjectWithMicro() {
	// check if manifest exists
	manifestExists, err := manifest.ManifestExists(dir)
	if err != nil {
		logger.Printf("Failed to check if a manifest file already exists, %v\n", err)
		return
	}
	if manifestExists {
		logger.Printf(`"deta.yml" already exists in the dir %s\n`, dir)
		return
	}

	logger.Printf("Creating new project \"%s\"...\n", name)
	m, err := manifest.CreateBlankManifest(dir)
	if err != nil {
		logger.Println(err)
		return
	}

	err = addMicro(&addMicroInput{manifest: m})
	if err != nil {
		logger.Printf("Error: failed to add a new micro, %v\n", err)
		return
	}
}

func new(cmd *cobra.Command, args []string) error {
	var err error

	if isFlagSet(name) && isFlagSet(dir) && confirmCreateProject {
		dir = filepath.Clean(dir)
		createBlankProject()
		return nil
	}

	if !isFlagSet(name) {
		name, err = selectProjectName()
		if err != nil {
			logger.Printf("problem while trying to retrieve micro's name through text prompt, %v\n", err)
			return nil
		}
	}

	if !isFlagSet(dir) {
		dir, err = selectDir()
		if err != nil {
			logger.Printf("problem while trying to retrieve micro's dir through text prompt, %v\n", err)
			return nil
		}
		dir = filepath.Clean(dir)
	}

	create, err := confirmCreateMicro()
	if err != nil {
		logger.Printf("problem while trying to get confirmation to create a new micro through confirm prompt, %v\n", err)
		return nil
	}

	if !create {
		createBlankProject()
		return nil
	}

	if create {
		createProjectWithMicro()
		return nil
	}

	return nil
}

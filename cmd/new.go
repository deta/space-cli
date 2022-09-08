package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/confirm"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/deta/pc-cli/pkg/util/fs"
	"github.com/spf13/cobra"
)

var (
	projectName string
	projectDir  string
	blank       bool

	newCmd = &cobra.Command{
		Use:   "new [flags]",
		Short: "create new project",
		RunE:  new,
	}
)

const (
	// blank template
	Blank string = "blank"
)

func init() {
	newCmd.Flags().StringVarP(&projectName, "name", "n", "", "project name")
	newCmd.Flags().StringVarP(&projectDir, "dir", "d", "./", "src of project to release")
	newCmd.Flags().BoolVarP(&blank, "blank", "b", false, "create blank project")
	rootCmd.AddCommand(newCmd)
}

func getDefaultAlias(projectName string) string {
	aliasRegexp := regexp.MustCompile(`/([^\w])/g,`)
	return aliasRegexp.ReplaceAllString(projectName, "")
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

/*
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
*/

func confirmCreateProjectWithDetectedConfig() (bool, error) {
	return confirm.Run(&confirm.Input{Prompt: "Do you want to create a project with the auto-detected configuration?"})
}

func createProject(name string, runtimeManager *runtime.Manager) error {
	res, err := client.CreateProject(&api.CreateProjectRequest{
		Name:  name,
		Alias: getDefaultAlias(name),
	})
	if err != nil {
		return err
	}

	err = runtimeManager.StoreProjectMeta(&runtime.ProjectMeta{ID: res.ID, Name: res.Name, Alias: res.Alias})
	if err != nil {
		return fmt.Errorf("failed to write project id to .space/meta, %w", err)
	}

	return nil
}

func new(cmd *cobra.Command, args []string) error {
	var err error

	if isFlagEmpty(projectName) {
		projectName, err = selectProjectName()
		if err != nil {
			return fmt.Errorf("problem while trying to get project's name through text prompt, %w", err)
		}
	}

	projectDir = filepath.Clean(projectDir)

	runtimeManager, err := runtime.NewManager(&projectDir, true)
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
		return fmt.Errorf("problem while trying to check contents of dir %s, %w", projectDir, err)
	}

	// create blank project if blank flag provided or if project folder is empty
	if blank || isEmpty {
		_, err = manifest.CreateBlankManifest(projectDir)
		if err != nil {
			return fmt.Errorf("failed to create blank project, %w", err)
		}

		logger.Printf("Creating blank project %s...\n", projectName)

		err = createProject(projectName, runtimeManager)
		if err != nil {
			return err
		}

		logger.Println(createProjectSuccessLogs(projectName))
		return nil
	}

	/*
		// prompt to start from template
		if isEmpty {
			// TODO: select template
			// TODO: download template files
			// TODO: improve successfull create project logs
		}
	*/

	isManifestPresent, err := manifest.IsManifestPresent(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to check for manifest file in dir %s, %w", projectDir, err)
	}

	// yes yaml
	if isManifestPresent {
		logger.Printf("Validating manifest...\n\n")

		m, err := manifest.Open(projectDir)
		if err != nil {
			logger.Printf("Error: %v\n", err)
			return nil
		}

		// validate manifest before creating new project with the existing manifest
		manifestErrors := scanner.ValidateManifest(m)

		if len(manifestErrors) > 0 {
			logValidationErrors(m, manifestErrors)
			logger.Println(styles.Error.Render("\nPlease try to fix the issues with manifest before creating a new project with the manifest."))
			return nil
		} else {
			logger.Printf("Nice! Manifest looks good 🎉!\n\n")
		}

		logger.Printf("Creating project \"%s\" with the existing manifest...\n", projectName)

		err = createProject(projectName, runtimeManager)
		if err != nil {
			return err
		}

		logger.Println(createProjectSuccessLogs(projectName))
		return nil
	}

	// no yaml present, auto-detect micros
	autoDetectedMicros, err := scanner.Scan(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to auto detect runtimes/frameworks for project %s, %w", projectName, err)
	}

	if len(autoDetectedMicros) > 0 {
		// prompt user for confirmation to create project with detected configuration
		logScannedMicros(autoDetectedMicros)
		create, err := confirmCreateProjectWithDetectedConfig()
		if err != nil {
			return fmt.Errorf("problem while trying to get confirmation to create project with the auto-detected configuration from confirm prompt, %w", err)
		}

		// create project with detected config
		if create {
			_, err = manifest.CreateManifestWithMicros(projectDir, autoDetectedMicros)
			if err != nil {
				return fmt.Errorf("failed to create project with detected micros, %w", err)
			}

			logger.Printf("Creating project %s with detected config....\n", projectName)
			err = createProject(projectName, runtimeManager)
			if err != nil {
				return err
			}

			logger.Println(createProjectSuccessLogs(projectName))
			return nil
		}
	}

	// don't create project with detected config, create blank project, point to docs
	_, err = manifest.CreateBlankManifest(projectDir)
	if err != nil {
		return fmt.Errorf("failed to create blank project, %w", err)
	}

	logger.Printf("Creating blank project %s...\n", projectName)
	err = createProject(projectName, runtimeManager)
	if err != nil {
		return err
	}

	logger.Println(createProjectSuccessLogs(projectName))

	return nil
}

func createProjectSuccessLogs(projectName string) string {
	logs := `
Project "%s" created successfully! https://deta.space/builder/%s
"space.yml", contains the configuration of your project & tells Deta how to build and run it.

Please modify your "space.yml" file to add micros. Here is a referece for information on adding more micros https://docs.deta.sh/manifest/add-micro

To push this project, run the command "deta push".`
	return fmt.Sprintf(logs, projectName, projectName)
}

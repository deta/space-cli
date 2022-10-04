package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/confirm"
	"github.com/deta/pc-cli/pkg/components/emoji"
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

func init() {
	newCmd.Flags().StringVarP(&projectName, "name", "n", "", "project name")
	newCmd.Flags().StringVarP(&projectDir, "dir", "d", "./", "src of project to release")
	newCmd.Flags().BoolVarP(&blank, "blank", "b", false, "create blank project")
	rootCmd.AddCommand(newCmd)
}

func getDefaultAlias(projectName string) string {
	aliasRegexp := regexp.MustCompile(`([^\w])`)
	return strings.ToLower(aliasRegexp.ReplaceAllString(projectName, ""))
}

func projectNameValidator(projectName string) error {

	if len(projectName) < 4 {
		return fmt.Errorf("project name \"%s\" must be at least 4 characters long", projectName)
	}

	if len(projectName) > 16 {
		return fmt.Errorf("project name \"%s\" must be at most 16 characters long", projectName)
	}

	return nil
}

func selectProjectName(placeholder string) (string, error) {

	promptInput := text.Input{
		Prompt:      "What is your project's name?",
		Placeholder: placeholder,
		Validator:   projectNameValidator,
	}

	return text.Run(&promptInput)
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
	logger.Println()
	var err error

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
		logger.Println(styles.Error("A project already exists in this directory. You can use"),
			styles.Code("deta push"), styles.Error("to create a Revision."))
		return nil
	}

	if isFlagEmpty(projectName) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		absWd, err := filepath.Abs(wd)
		if err != nil {
			return err
		}
		projectName = filepath.Base(absWd)

		projectName, err = selectProjectName(projectName)
		if err != nil {
			return fmt.Errorf("problem while trying to get project's name through prompt, %w", err)
		}
	}

	isEmpty, err := fs.IsEmpty(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to check contents of dir %s, %w", projectDir, err)
	}

	// create blank project if blank flag provided or if project folder is empty
	if blank || isEmpty {

		logger.Printf("%s No Spacefile found, trying to auto-detect configuration ...\n", emoji.Package)
		logger.Printf("%s Empty directory detected, creating %s from scratch ...\n", emoji.Package, styles.Pink(projectName))

		_, err = spacefile.CreateBlankSpacefile(projectDir)
		if err != nil {
			return fmt.Errorf("failed to create blank project, %w", err)
		}

		err = createProject(projectName, runtimeManager)
		if err != nil {
			return err
		}

		logger.Println(styles.Greenf("%s Project", emoji.Check), styles.Pink(projectName), styles.Green("created successfully!"))
		projectInfo, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return fmt.Errorf("failed to retrieve project info")
		}
		logger.Println(projectNotes(projectInfo.Name, projectInfo.ID))
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

	isSpacefilePresent, err := spacefile.IsSpacefilePresent(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to check for spacefile file in dir %s, %w", projectDir, err)
	}

	// Spacefile exists
	if isSpacefilePresent {
		logger.Printf("%s Spacefile found locally, validating Spacefile ...\n\n", emoji.Package)
		logger.Printf("Validating Spacefile ...\n\n")

		s, err := spacefile.Open(projectDir)
		if err != nil {
			logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
			return nil
		}

		// validate spacefile before creating new project with the existing spacefile
		spacefileErrors := scanner.ValidateSpacefile(s)

		if len(spacefileErrors) > 0 {
			logValidationErrors(s, spacefileErrors)
			logger.Println(styles.Errorf("Please fix the issues with your Spacefile before creating %s.\n", styles.Pink(projectName)))
			logger.Printf("The Spacefile documentation is here: %s", styles.Bold("https://go.deta.dev/docs/spacefile/v0"))

			return nil
		} else {
			logger.Printf("%s Nice, your Spacefile looks good!\n", emoji.PointDown)
		}

		logger.Printf("%s Creating project %s with your Spacefile ...\n", emoji.Package, styles.Pink(projectName))

		err = createProject(projectName, runtimeManager)
		if err != nil {
			return err
		}

		logger.Println(styles.Greenf("%s Project", emoji.Check), styles.Pink(projectName), styles.Green("created successfully!"))
		projectInfo, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return fmt.Errorf("failed to retrieve project info")
		}
		logger.Println(projectNotes(projectInfo.Name, projectInfo.ID))
		return nil
	}

	// no Spacefile present, auto-detect micros
	logger.Printf("%s No Spacefile found, trying to auto-detect configuration ...", emoji.Package)

	autoDetectedMicros, err := scanner.Scan(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to auto detect runtimes/frameworks for project %s, %w", projectName, err)
	}

	if len(autoDetectedMicros) > 0 {
		// prompt user for confirmation to create project with detected configuration
		logger.Printf("%s Space detected the following configuration:\n\n", emoji.PointDown)
		logDetectedMicros(autoDetectedMicros)

		create, err := confirm.Run(&confirm.Input{
			Prompt: fmt.Sprintf("Do you want to bootstrap %s with this configuration?", styles.Pink(projectName)),
		})
		if err != nil {
			return fmt.Errorf("problem while trying to get confirmation to create project with the auto-detected configuration from confirm prompt, %w", err)
		}

		// create project with detected config
		if create {
			logger.Printf("%s Bootstrapping %s ...\n", emoji.Package, styles.Pink(projectName))

			_, err = spacefile.CreateSpacefileWithMicros(projectDir, autoDetectedMicros)
			if err != nil {
				return fmt.Errorf("failed to create project with detected micros, %w", err)
			}

			err = createProject(projectName, runtimeManager)
			if err != nil {
				return err
			}

			logger.Println(styles.Greenf("%s Project", emoji.Check), styles.Pink(projectName), styles.Green("created successfully!"))
			projectInfo, err := runtimeManager.GetProjectMeta()
			if err != nil {
				return fmt.Errorf("failed to retrieve project info")
			}
			logger.Println(projectNotes(projectInfo.Name, projectInfo.ID))
			return nil
		}
	}

	// don't create project with detected config, create blank project, point to docs
	logger.Printf("%s Creating %s from scratch ...\n", emoji.Package, styles.Pink(projectName))

	_, err = spacefile.CreateBlankSpacefile(projectDir)
	if err != nil {
		return fmt.Errorf("failed to create blank project, %w", err)
	}

	err = createProject(projectName, runtimeManager)
	if err != nil {
		return err
	}

	logger.Println(styles.Greenf("%s Project", emoji.Check), styles.Pink(projectName), styles.Green("created successfully!"))
	projectInfo, err := runtimeManager.GetProjectMeta()
	if err != nil {
		return fmt.Errorf("failed to retrieve project info")
	}
	logger.Println(projectNotes(projectInfo.Name, projectInfo.ID))

	return nil
}

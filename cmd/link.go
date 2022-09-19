package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/confirm"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/pkg/scanner"
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
		Prompt:      "What is your Project ID of the project that you want to link to?",
		Placeholder: "",
		Validator:   projectIDValidator,
	}

	return text.Run(&promptInput)
}

func link(cmd *cobra.Command, args []string) error {
	logger.Println()
	var err error

	if isFlagEmpty(linkProjectID) {
		linkProjectID, err = selectLinkProjectID()
		if err != nil {
			return err
		}
	}

	linkProjectDir = filepath.Clean(linkProjectDir)

	runtimeManager, err := runtime.NewManager(&linkProjectDir, true)
	if err != nil {
		return err
	}

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
		return nil
	}

	isManifestPresent, err := manifest.IsManifestPresent(linkProjectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to scan manifest in dir %s, %v", linkProjectDir, err)
	}

	// yes yaml
	if isManifestPresent {
		logger.Printf("%s Space Manifest found, linking project with id \"%s\" to Space ...\n", emoji.Gear, linkProjectID)

		project, err := client.GetProject(&api.GetProjectRequest{ID: linkProjectID})
		if err != nil {
			if errors.Is(err, api.ErrProjectNotFound) {
				logger.Println(styles.Errorf("%s No project found. Please provide a valid Project ID.", emoji.ErrorExclamation))
				return nil
			}
			return err
		}

		err = runtimeManager.StoreProjectMeta(&runtime.ProjectMeta{ID: project.ID, Name: project.Name, Alias: project.Alias})
		if err != nil {
			return fmt.Errorf("failed to link project, %w", err)
		}

		logger.Println(styles.Greenf("%s Project", emoji.Link), styles.Pink(project.Name), styles.Green("was linked!"))
		projectInfo, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return fmt.Errorf("failed to retrieve project info")
		}
		logger.Println(projectNotes(projectInfo.Name, projectInfo.ID))
		return nil
	}

	// no yaml present, auto-detect micros
	logger.Printf("%s No Space Manifest found, trying to auto-detect configuration ...\n", emoji.Gear)
	autoDetectedMicros, err := scanner.Scan(linkProjectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to auto detect runtimes/frameworks, %v", err)
	}

	if len(autoDetectedMicros) > 0 {
		// prompt user for confirmation to link project with detected configuration
		logger.Printf("%s Deta detected the following configuration:\n\n", emoji.PointDown)
		logDetectedMicros(autoDetectedMicros)

		link, err := confirm.Run(&confirm.Input{
			Prompt: "Do you want to use this configuration?",
		})
		if err != nil {
			return fmt.Errorf("problem while trying to get confirmation to link project with the auto-detected configuration from prompt, %v", err)
		}

		// link project with detected config
		if link {
			logger.Printf("%s Linking project with ID \"%s\" using bootstrapped configuration ...\n", emoji.Gear, linkProjectID)

			project, err := client.GetProject(&api.GetProjectRequest{ID: linkProjectID})
			if err != nil {
				if errors.Is(err, api.ErrProjectNotFound) {
					logger.Println(styles.Error(fmt.Sprintf("%s No project found. Please provide a valid Project ID.", emoji.ErrorExclamation)))
					return nil
				}
				return err
			}

			_, err = manifest.CreateManifestWithMicros(linkProjectDir, autoDetectedMicros)
			if err != nil {
				return fmt.Errorf("failed to link project with detected micros, %w", err)
			}

			// TODO: verify project id through request
			err = runtimeManager.StoreProjectMeta(&runtime.ProjectMeta{ID: linkProjectID})
			if err != nil {
				return fmt.Errorf("failed to link project, %w", err)
			}

			logger.Println(styles.Greenf("%s Project", emoji.Link), styles.Pink(project.Name), styles.Green("was linked!"))
			projectInfo, err := runtimeManager.GetProjectMeta()
			if err != nil {
				return fmt.Errorf("failed to retrieve project info")
			}
			logger.Println(projectNotes(projectInfo.Name, projectInfo.ID))
			return nil
		}
	}

	// linking with blank
	logger.Printf("%s Linking project with id \"%s\" with a blank configuration ...\n", emoji.Gear, linkProjectID)

	project, err := client.GetProject(&api.GetProjectRequest{ID: linkProjectID})
	if err != nil {
		if errors.Is(err, api.ErrProjectNotFound) {
			logger.Println(styles.Errorf("%s No project found. Please provide a valid Project ID.", emoji.ErrorExclamation))
			return nil
		}
		return err
	}

	_, err = manifest.CreateBlankManifest(linkProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create blank project, %w", err)
	}

	// TODO: verify project id through request
	err = runtimeManager.StoreProjectMeta(&runtime.ProjectMeta{ID: linkProjectID})
	if err != nil {
		return fmt.Errorf("failed to link project, %w", err)
	}

	logger.Println(styles.Greenf("%s Project", emoji.Link), styles.Pink(project.Name), styles.Green("was linked!"))
	projectInfo, err := runtimeManager.GetProjectMeta()
	if err != nil {
		return fmt.Errorf("failed to retrieve project info")
	}
	logger.Println(projectNotes(projectInfo.Name, projectInfo.ID))
	return nil
}

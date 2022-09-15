package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/confirm"
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
		logger.Printf("🤠 This directory is already linked to a project named \"%s\".\n", existingProjectMeta.Name)
		logger.Println(projectNotes(existingProjectMeta.Name))
		return nil
	}

	isManifestPresent, err := manifest.IsManifestPresent(linkProjectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to scan manifest in dir %s, %v", linkProjectDir, err)
	}

	// yes yaml
	if isManifestPresent {
		logger.Printf("⚙️ Space Manifest found, linking project with id \"%s\" to Space ...\n", linkProjectID)

		project, err := client.GetProject(&api.GetProjectRequest{ID: linkProjectID})
		if err != nil {
			if errors.Is(err, api.ErrProjectNotFound) {
				logger.Println("❗ No project found. Please provide a valid Project ID.")
				return nil
			}
			return err
		}

		err = runtimeManager.StoreProjectMeta(&runtime.ProjectMeta{ID: project.ID, Name: project.Name, Alias: project.Alias})
		if err != nil {
			return fmt.Errorf("failed to link project, %w", err)
		}

		logger.Printf("🔗 Project \"%s\" was linked!\n", project.Name)
		logger.Println(projectNotes(project.Name))
		return nil
	}

	// no yaml present, auto-detect micros
	logger.Println("⚙️ No Space Manifest found, trying to auto-detect configuration ...")
	autoDetectedMicros, err := scanner.Scan(linkProjectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to auto detect runtimes/frameworks, %v", err)
	}

	if len(autoDetectedMicros) > 0 {
		// prompt user for confirmation to link project with detected configuration
		logger.Printf("👇 Deta detected the following configuration:\n\n")
		logDetectedMicros(autoDetectedMicros)

		link, err := confirm.Run(&confirm.Input{
			Prompt: "Do you want to use this configuration?",
		})
		if err != nil {
			return fmt.Errorf("problem while trying to get confirmation to link project with the auto-detected configuration from prompt, %v", err)
		}

		// link project with detected config
		if link {
			logger.Printf("⚙️ Linking project with ID \"%s\" using bootstrapped configuration ...\n", linkProjectID)

			project, err := client.GetProject(&api.GetProjectRequest{ID: linkProjectID})
			if err != nil {
				if errors.Is(err, api.ErrProjectNotFound) {
					logger.Println("No project found. Please provide a valid Project ID.")
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

			logger.Printf("🔗 Project \"%s\" was linked!\n", project.Name)
			logger.Println(projectNotes(project.Name))
			return nil
		}
	}

	// linking with blank
	logger.Printf("⚙️ Linking project with id \"%s\" with a blank configuration ...\n", linkProjectID)

	project, err := client.GetProject(&api.GetProjectRequest{ID: linkProjectID})
	if err != nil {
		if errors.Is(err, api.ErrProjectNotFound) {
			logger.Println("No project found. Please provide a valid Project ID.")
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

	logger.Printf("🔗 Project \"%s\" was linked!\n", project.Name)
	logger.Println(projectNotes(project.Name))
	return nil
}

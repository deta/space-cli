package cmd

import (
	"fmt"
	"path/filepath"

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
		Prompt:      "What's the project id of the project that you want to link?",
		Placeholder: "",
		Validator:   projectIDValidator,
	}

	return text.Run(&promptInput)
}

func confirmLinkProjectWithDetectedConfig() (bool, error) {
	return confirm.Run(&confirm.Input{Prompt: "Do you want to link to a project with the auto-detected configuration?"})
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

	runtimeManager, err := runtime.NewManager(&linkProjectDir, false)
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

	isManifestPresent, err := manifest.IsManifestPresent(linkProjectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to scan manifest in dir %s, %v", linkProjectDir, err)
	}

	// yes yaml
	if isManifestPresent {
		logger.Printf(`Linking project with id "%s"...`, linkProjectID)
		// TODO: verify project id through request
		err = runtimeManager.StoreProjectMeta(&runtime.ProjectMeta{ID: linkProjectID})
		if err != nil {
			return fmt.Errorf("failed to link project, %w", err)
		}
		logger.Println("Successfully linked project!")
		return nil
	}

	// no yaml present, auto-detect micros
	autoDetectedMicros, err := scanner.Scan(linkProjectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to auto detect runtimes/frameworks, %v", err)
	}

	if len(autoDetectedMicros) > 0 {
		// prompt user for confirmation to link project with detected configuration
		logScannedMicros(autoDetectedMicros)
		link, err := confirmLinkProjectWithDetectedConfig()
		if err != nil {
			return fmt.Errorf("problem while trying to get confirmation to link project with the auto-detected configuration from confirm prompt, %v", err)
		}

		// link project with detected config
		if link {
			_, err = manifest.CreateManifestWithMicros(linkProjectDir, autoDetectedMicros)
			if err != nil {
				return fmt.Errorf("failed to link project with detected micros, %w", err)
			}

			logger.Printf("Linking project with id %s with detected config....\n", linkProjectID)
			// TODO: verify project id through request
			err = runtimeManager.StoreProjectMeta(&runtime.ProjectMeta{ID: linkProjectID})
			if err != nil {
				return fmt.Errorf("failed to link project, %w", err)
			}
			logger.Println("Successfully linked project!")
			return nil
		}
	}

	// linking with blank
	_, err = manifest.CreateBlankManifest(linkProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create blank project, %w", err)
	}

	logger.Printf("Linking project with id %s with blank project...\n", linkProjectID)
	// TODO: verify project id through request
	err = runtimeManager.StoreProjectMeta(&runtime.ProjectMeta{ID: linkProjectID})
	if err != nil {
		return fmt.Errorf("failed to link project, %w", err)
	}
	logger.Println("Successfully linked project!")
	return nil
}

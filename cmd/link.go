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
	linkProjectId  string
	linkProjectDir string
	linkCmd        = &cobra.Command{
		Use:   "link [flags]",
		Short: "link code",
		RunE:  link,
	}
)

func init() {
	linkCmd.Flags().StringVarP(&linkProjectId, "id", "i", "", "what's your project id?")
	linkCmd.Flags().StringVarP(&linkProjectDir, "dir", "d", "./", "where's the project you want to link?")
	rootCmd.AddCommand(linkCmd)
}

func selectLinkProjectId() (string, error) {
	promptInput := text.Input{
		Prompt:      "What's the project id of the project that you want to link?",
		Placeholder: "",
		Validator:   projectIdValidator,
	}

	return text.Run(&promptInput)
}

func confirmLinkProjectWithDetectedConfig() (bool, error) {
	return confirm.Run(&confirm.Input{Prompt: "Do you want to link to a project with the auto-detected configuration?"})
}

func link(cmd *cobra.Command, args []string) error {
	var err error

	if isFlagEmpty(linkProjectId) {
		linkProjectId, err = selectLinkProjectId()
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
		logger.Printf(`Linking project with id "%s"...`, linkProjectId)
		// TODO: verify project id through request
		// TODO: write project id to .space/meta
		return nil
	}

	// no yaml present

	// auto-detect micros
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
			logger.Printf("Linking project with id %s with detected config....\n", linkProjectId)
			// TODO: verify project id through request
			// TODO: write project id to .space/meta
			return nil
		}
	}

	// don't link project with blank config, link blank project, point to docs
	logger.Printf("Linking project with id %s with blank project...\n", linkProjectId)
	logger.Println("Read docs...")
	// TODO: verify project id through request
	// TODO: write project id to .space/meta
	return nil
}

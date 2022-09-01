package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	validateDir string
	validateCmd = &cobra.Command{
		Use:   "validate [flags]",
		Short: "validate micros in dir",
		RunE:  validate,
	}
)

func init() {
	validateCmd.Flags().StringVarP(&validateDir, "dir", "d", "./", "where's the project you want to validate?")
	rootCmd.AddCommand(validateCmd)
}

func validate(cmd *cobra.Command, args []string) error {

	var valid bool = true

	validateDir = filepath.Clean(validateDir)

	isManifestPresent, err := manifest.IsManifestPresent(validateDir)
	if err != nil {
		return fmt.Errorf("problem while trying to scan manifest in the dir %s, %w", validateDir, err)
	}

	if !isManifestPresent {
		logger.Printf("No manifest file found in dir: %s to validate\n", validateDir)
		return nil
	}

	manifest, err := manifest.Open(validateDir)
	if err != nil {
		return fmt.Errorf("problem while opening manifest in dir %s, %w", validateDir, err)
	}

	// basic validation, check for path's of icon, micros and make sure they exist
	err = scanner.ValidateManifestIcon(manifest)
	if errors.Is(scanner.ErrInvalidIcon, err) {
		valid = false
		logger.Println("Cannot find icon path. Please provide a valid icon path or leave it empty to auto-generate project icon.")
	}

	logger.Println("Scanning micros...")
	for _, micro := range manifest.Micros {
		microLog := fmt.Sprintf("%s\n", micro.Name)
		microLog += fmt.Sprintf("L src: %s\n", micro.Src)
		microLog += fmt.Sprintf("L engine: %s", micro.Engine)
		logger.Println(microLog)

		err = scanner.ValidateMicro(micro)

		if err != nil {
			valid = false
		}

		if errors.Is(scanner.ErrEmptyMicroName, err) {
			logger.Println("L Error: Empty micro name. Please provide a valid name (cannot be empty).")
		}

		if errors.Is(scanner.ErrEmptyMicroSrc, err) {
			logger.Println("L Error: Empty micro src. Please provide a valid src for micro.")
		}

		if errors.Is(scanner.ErrEmptyMicroEngine, err) {
			logger.Println("L Error: Empty micro engine. Please provide a valid engine for micro.")
		}

		if errors.Is(scanner.ErrInvalidMicroSrc, err) {
			logger.Println("L Error: Cannot find src for micro. Please provide a valid src for where the micro exists.")
		}
	}

	if valid {
		logger.Println("Manifest looks ✨!")
	} else {
		logger.Println("Detected some issues with the manifest as stated earlier. Please try to fix them before pushing your code.")
	}

	return nil
}

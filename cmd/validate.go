package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	validateDir string
	validateCmd = &cobra.Command{
		Use:   "validate [flags]",
		Short: "validate manifest in dir",
		RunE:  validate,
	}
)

func init() {
	validateCmd.Flags().StringVarP(&validateDir, "dir", "d", "./", "src of project to validate")
	rootCmd.AddCommand(validateCmd)
}

// logValidationErrors logs manifest validation errors
func logValidationErrors(m *manifest.Manifest, manifestErrors []error) {
	// micro specfic errors
	microErrors := map[string][]error{}
	for _, err := range manifestErrors {
		if microError, ok := err.(*scanner.MicroError); ok {
			// filter micro specific errors
			micro := microError.Micro
			microErrors[micro.Name] = append(microErrors[micro.Name], microError.Err)
		} else {
			// general errors
			switch {
			case errors.Is(scanner.ErrExceedsMaxMicroLimit, err):
				logger.Println(styles.Error.Render("❌ Manifest exceeds max micro limit. Please make sure to use a max of 5 micros."))
			case errors.Is(scanner.ErrDuplicateMicros, err):
				logger.Println(styles.Error.Render("❌ Duplicate micro names. Please make sure to use unique names for micros."))
			case errors.Is(scanner.ErrNoPrimaryMicro, err):
				logger.Println(styles.Error.Render("❌ No primary micro specified. Please mark one of the micros as primary."))
			case errors.Is(scanner.ErrInvalidIcon, err):
				logger.Println(styles.Error.Render("❌ Cannot find icon path. Please provide a valid icon path or leave it empty to auto-generate project icon."))
			default:
				logger.Println(styles.Error.Render(fmt.Sprintf("❌ Error: %v", err)))
			}
		}
	}

	// basic validation, check src of micros and make sure they exist, invalid names/engines
	if len(m.Micros) > 0 {
		logger.Println(styles.Green.Render("\nScanned micros:"))
	}

	for _, micro := range m.Micros {
		logMicro(micro)

		microErrors := microErrors[micro.Name]
		logger.Println("Errors:")
		if len(microErrors) == 0 {
			logger.Println(styles.Green.Render("✔ No errors detected for this micro."))
		}
		for _, err := range microErrors {
			switch {
			case errors.Is(scanner.ErrEmptyMicroName, err):
				logger.Println(styles.Error.Render("❌ Empty micro name. Please provide a valid name (cannot be empty)."))
			case errors.Is(scanner.ErrEmptyMicroSrc, err):
				logger.Println(styles.Error.Render("❌ Empty micro src. Please provide a valid src for micro."))
			case errors.Is(scanner.ErrEmptyMicroEngine, err):
				logger.Println(styles.Error.Render("❌ Empty micro engine. Please provide a valid engine for micro."))
			case errors.Is(scanner.ErrInvalidMicroSrc, err):
				logger.Println(styles.Error.Render("❌ Cannot find src for micro. Please provide a valid src for where the micro exists."))
			case errors.Is(scanner.ErrInvalidMicroEngine, err):
				logger.Println(styles.Error.Render("❌ Invalid engine. Please check the docs for all the supported engines."))
			default:
				logger.Println(styles.Error.Render(fmt.Sprintf("❌ Micro Error: %v", err)))
			}
		}
	}
}

func validate(cmd *cobra.Command, args []string) error {

	validateDir = filepath.Clean(validateDir)

	isManifestPresent, err := manifest.IsManifestPresent(validateDir)
	if err != nil {
		return fmt.Errorf("problem while trying to scan manifest in the dir %s, %w", validateDir, err)
	}

	if !isManifestPresent {
		logger.Printf("No manifest file found in dir %s to validate\n", validateDir)
		return nil
	}

	m, err := manifest.Open(validateDir)
	if err != nil {
		return fmt.Errorf("problem while opening manifest in dir %s, %w", validateDir, err)
	}

	manifestErrors := scanner.ValidateManifest(m)
	logValidationErrors(m, manifestErrors)

	if len(manifestErrors) == 0 {
		logger.Println(styles.Green.Render("\nNice! Manifest looks good ✨!"))
	} else {
		logger.Println(styles.Error.Render("\nDetected some issues with the manifest. Please try to fix them before pushing your code."))
	}
	return nil
}

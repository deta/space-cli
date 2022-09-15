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

	var isIconValid bool = true

	for _, err := range manifestErrors {
		if microError, ok := err.(*scanner.MicroError); ok {
			// filter micro specific errors
			micro := microError.Micro
			microErrors[micro.Name] = append(microErrors[micro.Name], microError.Err)
		} else {
			// general errors
			switch {
			case errors.Is(scanner.ErrExceedsMaxMicroLimit, err):
				logger.Println(styles.Error("❌ Validation Error: Manifest exceeds max micro limit. Please make sure to use a max of 5 micros."))
			case errors.Is(scanner.ErrDuplicateMicros, err):
				logger.Println(styles.Error("❌ Validation Error: Duplicate micro names. Please make sure to use unique names for micros."))
			case errors.Is(scanner.ErrNoPrimaryMicro, err):
				logger.Println(styles.Error("❌ Validation Error: No primary micro specified. Please mark one of the micros as primary."))
			case errors.Is(scanner.ErrInvalidIcon, err):
				isIconValid = false
				logger.Println(styles.Error("❌ \"icon\": Cannot find icon path. Please provide a valid icon path or leave it empty to auto-generate project icon."))
			default:
				logger.Println(styles.Error(fmt.Sprintf("❌ Validation Error: %v", err)))
			}
		}
	}

	if isIconValid {
		logger.Printf("✅ Icon")
	}

	for _, micro := range m.Micros {
		microErrors := microErrors[micro.Name]
		if len(microErrors) == 0 {
			logger.Printf("✅ Micro \"%s\"\n", micro.Name)
		} else {
			msg := "\n❌ Micro"
			if micro.Name != "" {
				msg = fmt.Sprintf("%s %s:", msg, micro.Name)
			} else if micro.Src != "" {
				msg = fmt.Sprintf("%s with src \"%s/\":", msg, micro.Src)
			} else {
				msg = "\n❌ Invalid Micro"
			}
			logger.Println(msg)
		}

		for _, err := range microErrors {
			switch {
			case errors.Is(scanner.ErrEmptyMicroName, err):
				logger.Println(styles.Error("L Missing \"name\"\n"))
			case errors.Is(scanner.ErrEmptyMicroSrc, err):
				logger.Println(styles.Error("L Missing \"src\"\n"))
			case errors.Is(scanner.ErrEmptyMicroEngine, err):
				logger.Println(styles.Error("L Missing \"engine\"\n"))
			case errors.Is(scanner.ErrInvalidMicroSrc, err):
				logger.Println(styles.Error(fmt.Sprintf("L Cannot find src for micro \"%s\"\n", micro.Src)))
			case errors.Is(scanner.ErrInvalidMicroEngine, err):
				logger.Println(styles.Error(fmt.Sprintf("L Invalid engine value \"%s\"\n", micro.Src)))
			default:
				logger.Println(styles.Error(fmt.Sprintf("L %v", err)))
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
		logger.Println("No Space Manifest found in your directory.")
		return nil
	}

	logger.Printf("⚙️ Validating Space Manifest file ...\n\n")

	m, err := manifest.Open(validateDir)
	if err != nil {
		return fmt.Errorf("problem while opening manifest in dir %s, %w", validateDir, err)
	}

	manifestErrors := scanner.ValidateManifest(m)

	logValidationErrors(m, manifestErrors)

	if len(manifestErrors) == 0 {
		logger.Println(styles.Green("\n✨ Manifest looks good!"))
	} else {
		logger.Println(styles.Error("\n❗ Detected some issues with your Space Manifest. Please fix them before pushing your code."))
	}
	return nil
}

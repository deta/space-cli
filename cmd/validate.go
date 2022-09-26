package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	validateDir string
	validateCmd = &cobra.Command{
		Use:   "validate [flags]",
		Short: "validate spacefile in dir",
		RunE:  validate,
	}
)

func init() {
	validateCmd.Flags().StringVarP(&validateDir, "dir", "d", "./", "src of project to validate")
	rootCmd.AddCommand(validateCmd)
}

// logValidationErrors logs spacefile validation errors
func logValidationErrors(s *spacefile.Spacefile, spacefileErrors []error) {

	// micro specfic errors
	microErrors := map[string][]error{}

	var isIconValid bool = true

	for _, err := range spacefileErrors {
		if microError, ok := err.(*scanner.MicroError); ok {
			// filter micro specific errors
			micro := microError.Micro
			microErrors[micro.Name] = append(microErrors[micro.Name], microError.Err)
		} else {
			// general errors
			switch {
			case errors.Is(scanner.ErrExceedsMaxMicroLimit, err):
				logger.Println(styles.Errorf("%s Validation Error: Spacefile exceeds max micro limit. Please make sure to use a max of 5 micros.\n", emoji.X))
			case errors.Is(scanner.ErrDuplicateMicros, err):
				logger.Println(styles.Errorf("%s Validation Error: Duplicate micro names. Please make sure to use unique names for micros.\n", emoji.X))
			case errors.Is(scanner.ErrNoPrimaryMicro, err):
				logger.Println(styles.Errorf("%s Validation Error: No primary micro specified. Please mark one of the micros as primary.\n", emoji.X))
			case errors.Is(scanner.ErrInvalidIcon, err):
				isIconValid = false
				logger.Println(styles.Errorf("%s \"icon\": Cannot find icon path. Please provide a valid icon path or leave it empty to auto-generate project icon.\n", emoji.X))
			default:
				logger.Println(styles.Error(fmt.Sprintf("%s Validation Error: %v", emoji.X, err)))
			}
		}
	}

	if isIconValid {
		logger.Printf("%s Icon", emoji.Check)
	}

	for _, micro := range s.Micros {
		microErrors := microErrors[micro.Name]
		if len(microErrors) == 0 {
			logger.Printf("%s Micro \"%s\"\n", emoji.Check, micro.Name)
		} else {
			msg := fmt.Sprintf("\n%s Micro", emoji.X)
			if micro.Name != "" {
				msg = fmt.Sprintf("%s %s:", msg, micro.Name)
			} else if micro.Src != "" {
				msg = fmt.Sprintf("%s with src \"%s/\":", msg, micro.Src)
			} else {
				msg = fmt.Sprintf("\n%s Invalid Micro", emoji.X)
			}
			logger.Println(msg)
		}

		for _, err := range microErrors {
			switch {
			case errors.Is(scanner.ErrEmptyMicroName, err):
				logger.Println(styles.Error("L Missing \"name\""))
			case errors.Is(scanner.ErrEmptyMicroSrc, err):
				logger.Println(styles.Error("L Missing \"src\""))
			case errors.Is(scanner.ErrEmptyMicroEngine, err):
				logger.Println(styles.Error("L Missing \"engine\""))
			case errors.Is(scanner.ErrInvalidMicroSrc, err):
				logger.Println(styles.Error(fmt.Sprintf("L Cannot find src for micro \"%s\"", micro.Src)))
			case errors.Is(scanner.ErrInvalidMicroEngine, err):
				logger.Println(styles.Error(fmt.Sprintf("L Invalid engine value \"%s\"", micro.Src)))
			default:
				logger.Println(styles.Error(fmt.Sprintf("L Error: %v", err)))
			}
		}

		if len(microErrors) > 0 {
			logger.Println()
		}
	}
}

func validate(cmd *cobra.Command, args []string) error {
	logger.Println()
	validateDir = filepath.Clean(validateDir)

	isSpacefilePresent, err := spacefile.IsSpacefilePresent(validateDir)
	if err != nil {
		return fmt.Errorf("problem while trying to scan spacefile in the dir %s, %w", validateDir, err)
	}

	if !isSpacefilePresent {
		logger.Println(styles.Errorf("%s No Spacefile found in your directory.", emoji.ErrorExclamation))
		return nil
	}

	logger.Printf("%s Validating Spacefile file ...\n\n", emoji.Package)

	s, err := spacefile.Open(validateDir)
	if err != nil {
		return fmt.Errorf("problem while opening spacefile in dir %s, %w", validateDir, err)
	}

	spacefileErrors := scanner.ValidateSpacefile(s)

	logValidationErrors(s, spacefileErrors)

	if len(spacefileErrors) == 0 {
		logger.Println(styles.Greenf("\n%s Spacefile looks good!", emoji.Sparkles))
	} else {
		logger.Println(styles.Errorf("\n%s Detected some issues with your Spacefile. Please fix them before pushing your code.", emoji.ErrorExclamation))
	}
	return nil
}

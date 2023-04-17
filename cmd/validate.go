package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/spf13/cobra"
)

func newCmdValidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "validate [flags]",
		Short:    "Validate your Spacefile and check for errors",
		PostRunE: shared.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")
			if err := validate(projectDir); err != nil {
				return err
			}

			return nil
		},
		PreRunE: shared.CheckExists("dir"),
	}
	cmd.Flags().StringP("dir", "d", "./", "src of project to validate")

	return cmd
}

func validate(projectDir string) error {
	shared.Logger.Printf("\n%s Validating Spacefile...", emoji.Package)

	s, err := spacefile.ParseSpacefile(filepath.Join(projectDir, "Spacefile"))
	if err != nil {
		shared.Logger.Println(styles.Errorf("\n%s Detected some issues with your Spacefile. Please fix them before pushing your code.", emoji.ErrorExclamation))
		shared.Logger.Println()
		shared.Logger.Println(err.Error())
		return err
	}

	if s.Icon == "" {
		shared.Logger.Printf("\n%s No app icon specified.", styles.Blue("i"))
	} else {
		if err := spacefile.ValidateIcon(s.Icon); err != nil {
			shared.Logger.Println(styles.Errorf("\nDetected some issues with your icon. Please fix them before pushing your code."))
			switch {
			case errors.Is(spacefile.ErrInvalidIconType, err):
				shared.Logger.Println(styles.Error("L Invalid icon type. Please use a 512x512 sized PNG or WebP icon"))
			case errors.Is(spacefile.ErrInvalidIconSize, err):
				shared.Logger.Println(styles.Error("L Icon size is not valid. Please use a 512x512 sized PNG or WebP icon"))
			case errors.Is(spacefile.ErrInvalidIconPath, err):
				shared.Logger.Println(styles.Error("L Cannot find icon path. Please provide a valid icon path or leave it empty to auto-generate project icon."))
			default:
				shared.Logger.Println(styles.Error(fmt.Sprintf("%s Validation Error: %v", emoji.X, err)))
			}
			return err
		}
	}

	shared.Logger.Println(styles.Greenf("\n%s Spacefile looks good!", emoji.Sparkles))
	return nil
}

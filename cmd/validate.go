package cmd

import (
	"errors"
	"fmt"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/spf13/cobra"
)

func newCmdValidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "validate [flags]",
		Short:    "Validate your Spacefile and check for errors",
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")
			if err := validate(projectDir); err != nil {
				return err
			}

			return nil
		},
		PreRunE: utils.CheckExists("dir"),
	}
	cmd.Flags().StringP("dir", "d", "./", "src of project to validate")

	return cmd
}

func validate(projectDir string) error {
	utils.Logger.Printf("\n%s Validating Spacefile...", emoji.Package)

	s, err := spacefile.LoadSpacefile(projectDir)
	if err != nil {
		utils.Logger.Println(styles.Errorf("\n%s Detected some issues with your Spacefile. Please fix them before pushing your code.", emoji.ErrorExclamation))
		utils.Logger.Println()
		utils.Logger.Println(err.Error())
		return err
	}

	if s.Icon == "" {
		utils.Logger.Printf("\n%s No app icon specified.", styles.Blue("i"))
	} else {
		if err := spacefile.ValidateIcon(s.Icon); err != nil {
			utils.Logger.Println(styles.Errorf("\nDetected some issues with your icon. Please fix them before pushing your code."))
			switch {
			case errors.Is(spacefile.ErrInvalidIconType, err):
				utils.Logger.Println(styles.Error("L Invalid icon type. Please use a 512x512 sized PNG or WebP icon"))
			case errors.Is(spacefile.ErrInvalidIconSize, err):
				utils.Logger.Println(styles.Error("L Icon size is not valid. Please use a 512x512 sized PNG or WebP icon"))
			case errors.Is(spacefile.ErrInvalidIconPath, err):
				utils.Logger.Println(styles.Error("L Cannot find icon path. Please provide a valid icon path or leave it empty to auto-generate project icon."))
			default:
				utils.Logger.Println(styles.Error(fmt.Sprintf("%s Validation Error: %v", emoji.X, err)))
			}
			return err
		}
	}

	utils.Logger.Println(styles.Greenf("\n%s Spacefile looks good!", emoji.Sparkles))
	return nil
}

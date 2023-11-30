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
	utils.Logger.Printf("\n%s Validating your Spacefile...\n", emoji.Package)

	s, err := spacefile.LoadSpacefile(projectDir)
	if err != nil {
		return fmt.Errorf("failed to parse Spacefile, %w", err)
	}

	if s.Icon == "" {
		utils.Logger.Printf("\n%s No app icon specified.", styles.Blue("i"))
	} else {
		if err := spacefile.ValidateIcon(s.Icon); err != nil {
			switch {
			case errors.Is(spacefile.ErrInvalidIconType, err):
				return fmt.Errorf("invalid icon type, please use a 512x512 sized PNG or WebP icon")
			case errors.Is(spacefile.ErrInvalidIconSize, err):
				return fmt.Errorf("icon size is not valid, please use a 512x512 sized PNG or WebP icon")
			case errors.Is(spacefile.ErrInvalidIconPath, err):
				return fmt.Errorf("cannot find the icon in provided path, please provide a valid icon path or leave it empty to auto-generate one")
			default:
				return err
			}
		}
	}
	utils.Logger.Println(styles.Greenf("\n%s Spacefile looks good!", emoji.Sparkles))
	return nil
}

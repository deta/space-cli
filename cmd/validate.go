package cmd

import (
	"os"
	"path/filepath"

	"github.com/deta/pc-cli/cmd/shared"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/spf13/cobra"
)

func newCmdValidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [flags]",
		Short: "validate spacefile in dir",
		Run: func(cmd *cobra.Command, args []string) {
			projectDir, _ := cmd.Flags().GetString("dir")
			if err := validate(projectDir); err != nil {
				os.Exit(1)
			}
		},
		PreRunE: shared.CheckExists("dir"),
	}
	cmd.Flags().StringP("dir", "d", "./", "src of project to validate")

	return cmd
}

func validate(projectDir string) error {
	shared.Logger.Printf("\n%s Validating Spacefile...", emoji.Package)

	s, err := spacefile.Open(filepath.Join(projectDir, "Spacefile"))
	if err != nil {
		shared.Logger.Println(styles.Errorf("\n%s Detected some issues with your Spacefile. Please fix them before pushing your code.", emoji.ErrorExclamation))
		shared.Logger.Println()
		shared.Logger.Println(styles.Error(err.Error()))
		return err
	}

	if s.Icon == "" {
		shared.Logger.Printf("\n%s No app icon specified.", styles.Blue("i"))
	}

	shared.Logger.Println(styles.Greenf("\n%s Spacefile looks good!", emoji.Sparkles))
	return nil
}

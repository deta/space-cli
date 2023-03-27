package cmd

import (
	"os"
	"path/filepath"

	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/spf13/cobra"
)

func newCmdValidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "validate [flags]",
		Short:   "validate spacefile in dir",
		Run:     validate,
		PreRunE: CheckExists("dir"),
	}
	cmd.Flags().StringP("dir", "d", "./", "src of project to validate")

	return cmd
}

func validate(cmd *cobra.Command, args []string) {
	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	projectDir, _ := cmd.Flags().GetString("dir")

	logger.Printf("\n%s Validating Spacefile...", emoji.Package)

	s, err := spacefile.Parse(filepath.Join(projectDir, "Spacefile"))
	if err != nil {
		logger.Println(styles.Errorf("\n%s Detected some issues with your Spacefile. Please fix them before pushing your code.", emoji.ErrorExclamation))
		logger.Println()
		logger.Println(styles.Error(err.Error()))
		os.Exit(1)
	}

	if s.Icon == "" {
		logger.Printf("\n%s No app icon specified.", styles.Blue("i"))
	} else {
		if err := spacefile.ValidateIcon(s.Icon); err != nil {
			logger.Println(styles.Errorf("%s Detected some issues with your icon. Please fix them before pushing your code.", emoji.ErrorExclamation))
			logger.Println(styles.Error(err.Error()))
		}
	}

	logger.Println(styles.Greenf("\n%s Spacefile looks good!", emoji.Sparkles))

	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}
}

package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/deta/pc-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	client = api.NewDetaClient()
	logger = log.New(os.Stderr, "", 0)
)

func NewSpaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "space",
		Short: "Deta Space CLI for mananging Deta Space projects",
		Long: fmt.Sprintf(`Deta Space command line interface for managing Deta Space projects.
Complete documentation available at %s`, docsUrl),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
		// no usage shown on errors
		SilenceUsage:      false,
		DisableAutoGenTag: true,
	}

	cmd.AddCommand(newCmdLogin())
	cmd.AddCommand(newCmdLink())
	cmd.AddCommand(newCmdPush())
	cmd.AddCommand(newCmdExec())
	cmd.AddCommand(newCmdDev())
	cmd.AddCommand(newCmdNew())
	cmd.AddCommand(newCmdVersion())
	cmd.AddCommand(newCmdOpen())
	cmd.AddCommand(newCmdValidate())
	cmd.AddCommand(newCmdRelease())

	return cmd
}

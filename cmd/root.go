package cmd

import (
	"fmt"

	"github.com/deta/space/cmd/utils"
	"github.com/spf13/cobra"
)

func NewSpaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "space",
		Short: "Deta Space CLI",
		Long: fmt.Sprintf(`Deta Space command line interface for managing Deta Space projects.

Complete documentation available at %s`, utils.DocsUrl),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
		DisableAutoGenTag: true,
		Version:           utils.SpaceVersion,
	}

	cmd.AddCommand(newCmdLogin())
	cmd.AddCommand(newCmdLink())
	cmd.AddCommand(newCmdPush())
	cmd.AddCommand(newCmdExec())
	cmd.AddCommand(NewCmdDev())
	cmd.AddCommand(newCmdNew())
	cmd.AddCommand(NewCmdVersion(utils.SpaceVersion, utils.Platform))
	cmd.AddCommand(newCmdOpen())
	cmd.AddCommand(newCmdValidate())
	cmd.AddCommand(newCmdRelease())
	cmd.AddCommand(newCmdAPI())
	cmd.AddCommand(newCmdPrintAccessToken())
	cmd.AddCommand(newCmdTrigger())
	cmd.AddCommand(newCmdBuilder())
	cmd.AddCommand(newCmdPull())

	// XXX: This will prevent the usage from being displayed when an error occurs
	// while calling the Execute function in the main.go file.
	cmd.SilenceUsage = true

	return cmd
}

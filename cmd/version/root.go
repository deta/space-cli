package version

import (
	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/spf13/cobra"
)

func NewCmdVersion(version string, platform string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Space CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			shared.Logger.Printf("%s %s %s\n", emoji.Pistol, styles.Code(version), platform)
		},
	}

	cmd.AddCommand(newCmdVersionUpgrade(version))
	return cmd
}

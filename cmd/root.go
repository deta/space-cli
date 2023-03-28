package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/deta/pc-cli/cmd/dev"
	"github.com/deta/pc-cli/cmd/shared"
	"github.com/deta/pc-cli/cmd/version"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/spf13/cobra"
)

var (
	spaceVersion string = "dev"
	platform     string
)

func NewSpaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "space",
		Short: "Deta Space CLI for mananging Deta Space projects",
		Long: fmt.Sprintf(`Deta Space command line interface for managing Deta Space projects.
Complete documentation available at %s`, shared.DocsUrl),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
		// no usage shown on errors
		SilenceUsage:      false,
		DisableAutoGenTag: true,
		Version:           spaceVersion,
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if isPrerelease(spaceVersion) || spaceVersion == "dev" {
				return
			}

			latestVersion, lastCheck, err := auth.GetLatestCachedVersion()
			if err != nil || time.Since(lastCheck) > 24*time.Hour {
				res, err := shared.Client.GetLatestCLIVersion()
				if err != nil {
					return
				}

				auth.CacheLatestVersion(res.Tag)
				latestVersion = res.Tag
			}

			if spaceVersion != latestVersion {
				shared.Logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
			}
		},
	}

	cmd.AddCommand(newCmdLogin())
	cmd.AddCommand(newCmdLink())
	cmd.AddCommand(newCmdPush())
	cmd.AddCommand(newCmdExec())
	cmd.AddCommand(dev.NewCmdDev())
	cmd.AddCommand(newCmdNew())
	cmd.AddCommand(version.NewCmdVersion(spaceVersion, platform))
	cmd.AddCommand(newCmdOpen())
	cmd.AddCommand(newCmdValidate())
	cmd.AddCommand(newCmdRelease())

	return cmd
}

func isPrerelease(version string) bool {
	return len(strings.Split(version, "-")) > 1
}

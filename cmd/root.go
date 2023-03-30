package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/deta/space/cmd/dev"
	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/cmd/version"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/styles"
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
			if isPrerelease(spaceVersion) {
				return
			}

			latestVersion, lastCheck, err := runtime.GetLatestCachedVersion()
			if err != nil || time.Since(lastCheck) > 69*time.Minute {
				shared.Logger.Println("\nChecking for new Space CLI version...")
				res, err := shared.Client.GetLatestCLIVersion()
				if err != nil {
					shared.Logger.Println("Failed to check for new Space CLI version")
					return
				}

				runtime.CacheLatestVersion(res.Tag)
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

package cmd

import (
	"strings"

	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/spf13/cobra"
)

var (
	spaceVersion string
	platform     string

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Space CLI version",
		RunE:  version,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func version(cmd *cobra.Command, args []string) error {
	logger.Println()
	logger.Printf("%s %s %s\n", emoji.Pistol, styles.Code(spaceVersion), platform)

	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}

	return nil
}

type checkVersionMsg struct {
	isLower bool
	err     error
}

func isPrerelease(version string) bool {
	return len(strings.Split(version, "-")) > 1
}

func checkVersion(c chan *checkVersionMsg) {
	cm := &checkVersionMsg{}

	if isPrerelease(spaceVersion) {
		c <- cm
		return
	}

	latestVersion, err := client.GetLatestCLIVersion()
	if err != nil {
		cm.err = err
		c <- cm
		return
	}
	cm.isLower = spaceVersion != latestVersion.Tag && !latestVersion.Prerelease
	cm.err = nil
	c <- cm
}

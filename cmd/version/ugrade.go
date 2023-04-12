package version

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/api"
	detaruntime "github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/spf13/cobra"
)

func newCmdVersionUpgrade(currentVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade Space CLI version",
		Example: versionUpgradeExamples(),
		Run: func(cmd *cobra.Command, args []string) {
			targetVersion, _ := cmd.Flags().GetString("version")
			if !cmd.Flags().Changed("version") {
				latestVersion, err := api.GetLatestCliVersion()
				if err != nil {
					shared.Logger.Println(styles.Errorf("%s Failed to get latest version. Please try again.", emoji.X))
					os.Exit(1)
				}
				targetVersion = latestVersion
			}

			if currentVersion == targetVersion {
				shared.Logger.Println(styles.Boldf("Space CLI version already %s, no upgrade required", styles.Code(targetVersion)))
				return
			}

			switch runtime.GOOS {
			case "linux", "darwin":
				err := upgradeUnix(targetVersion)
				if err != nil {
					shared.Logger.Println(styles.Errorf("%s Upgrade failed. Please try again.", emoji.X))
					os.Exit(1)
				}
			case "windows":
				err := upgradeWin(targetVersion)
				if err != nil {
					shared.Logger.Println(styles.Errorf("%s Upgrade failed. Please try again.", emoji.X))
					os.Exit(1)
				}
			default:
				shared.Logger.Println(styles.Errorf("%s Upgrade not supported for %s", emoji.X, runtime.GOOS))
				os.Exit(1)
			}

			detaruntime.CacheLatestVersion(targetVersion)
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringP("version", "v", "", "version number")
	return cmd
}

func upgradeUnix(version string) error {
	curlCmd := exec.Command("curl", "-fsSL", "https://get.deta.dev/space-cli.sh")
	msg := "Upgrading Space CLI"
	curlOutput, err := curlCmd.CombinedOutput()
	if err != nil {
		shared.Logger.Println(string(curlOutput))
		return err
	}

	co := string(curlOutput)
	shCmd := exec.Command("sh", "-c", co)
	if version != "" {
		if !strings.HasPrefix(version, "v") {
			version = fmt.Sprintf("v%s", version)
		}
		msg = fmt.Sprintf("%s to version %s", msg, styles.Code(version))
		shCmd = exec.Command("sh", "-c", co, "upgrade", version)
	}
	shared.Logger.Printf("%s...\n", msg)

	shOutput, err := shCmd.CombinedOutput()
	shared.Logger.Println(string(shOutput))
	if err != nil {
		return err
	}
	return nil
}

func versionUpgradeExamples() string {
	return `
1. space version upgrade
Upgrade Space CLI to latest version.
2. space version upgrade --version v0.0.2
Upgrade Space CLI to version 'v0.0.2'.`
}

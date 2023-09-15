package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/api"
	detaruntime "github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/styles"
	"github.com/spf13/cobra"
)

func newCmdVersionUpgrade(currentVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade Space CLI version",
		Example: versionUpgradeExamples(),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetVersion, _ := cmd.Flags().GetString("version")
			if !cmd.Flags().Changed("version") {
				latestVersion, err := api.GetLatestCliVersion()
				if err != nil {
					return fmt.Errorf("failed to get the latest version, %w, please try again", err)
				}
				targetVersion = latestVersion
			}

			if currentVersion == targetVersion {
				utils.Logger.Println(styles.Boldf("Space CLI version already %s, no upgrade required", styles.Code(targetVersion)))
				return nil
			}

			switch runtime.GOOS {
			case "linux", "darwin":
				err := upgradeUnix(targetVersion)
				if err != nil {
					return fmt.Errorf("failed to upgrade, %w, please try again", err)
				}
			case "windows":
				err := upgradeWin(targetVersion)
				if err != nil {
					if err != nil {
						return fmt.Errorf("failed to upgrade, %w, please try again", err)
					}
				}
			default:
				return fmt.Errorf("unsupported OS, %s", runtime.GOOS)
			}

			detaruntime.CacheLatestVersion(targetVersion)

			return nil
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringP("version", "v", "", "version number")
	return cmd
}

func upgradeUnix(version string) error {
	curlCmd := exec.Command("curl", "-fsSL", "https://deta.space/assets/space-cli.sh")
	msg := "Upgrading Space CLI"
	curlOutput, err := curlCmd.CombinedOutput()
	if err != nil {
		utils.Logger.Println(string(curlOutput))
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
	utils.Logger.Printf("%s...\n", msg)

	shOutput, err := shCmd.CombinedOutput()
	utils.Logger.Println(string(shOutput))
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

package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/spf13/cobra"
)

func newCmdVersionUpgrade() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade Space CLI version",
		Example: versionUpgradeExamples(),
		RunE:    upgrade,
		Args:    cobra.NoArgs,
	}
	cmd.Flags().StringP("version", "v", "", "version number")
	return cmd
}

func upgrade(cmd *cobra.Command, args []string) error {
	logger.Println()
	version, _ := cmd.Flags().GetString("version")
	latestVersion, err := client.GetLatestCLIVersion()
	if err != nil {
		return err
	}

	upgradingTo := latestVersion.Tag
	if version != "" {
		if !strings.HasPrefix(version, "v") {
			version = fmt.Sprintf("v%s", version)
		}

		versionExists, err := client.CheckCLIVersionTag(version)
		if err != nil {
			logger.Println(styles.Errorf("%s Failed to check if version exists. Please check version and try again.", emoji.X))
			return nil
		}
		if !versionExists {
			logger.Println(styles.Errorf("%s not found.", styles.Code(version)))
			return nil
		}

		upgradingTo = version
	}
	if spaceVersion == upgradingTo {
		logger.Println(styles.Boldf("Space CLI version already %s, no upgrade required", styles.Code(upgradingTo)))
		return nil
	}

	switch runtime.GOOS {
	case "linux", "darwin":
		return upgradeUnix(version)
	case "windows":
		return upgradeWin(version)
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func upgradeUnix(version string) error {
	curlCmd := exec.Command("curl", "-fsSL", "https://get.deta.dev/space-cli.sh")
	msg := "Upgrading Space CLI"
	curlOutput, err := curlCmd.CombinedOutput()
	if err != nil {
		logger.Println(string(curlOutput))
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
	logger.Printf("%s...\n", msg)

	shOutput, err := shCmd.CombinedOutput()
	logger.Println(string(shOutput))
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

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

var (
	versionFlag string
	upgradeCmd  = &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade Space CLI version",
		Example: versionUpgradeExamples(),
		RunE:    upgrade,
		Args:    cobra.NoArgs,
	}
)

func init() {
	upgradeCmd.Flags().StringVarP(&versionFlag, "version", "v", "", "version number")
	versionCmd.AddCommand(upgradeCmd)
}

func upgrade(cmd *cobra.Command, args []string) error {
	logger.Println()
	latestVersion, err := client.GetLatestCLIVersion()
	if err != nil {
		return err
	}

	upgradingTo := latestVersion.Tag
	if versionFlag != "" {
		if !strings.HasPrefix(versionFlag, "v") {
			versionFlag = fmt.Sprintf("v%s", versionFlag)
		}

		versionExists, err := client.CheckCLIVersionTag(versionFlag)
		if err != nil {
			logger.Println(styles.Errorf("%s Failed to check if version exists. Please check version and try again.", emoji.X))
			return nil
		}
		if !versionExists {
			logger.Println(styles.Errorf("%s not found.", styles.Code(versionFlag)))
			return nil
		}

		upgradingTo = versionFlag
	}
	if spaceVersion == upgradingTo {
		logger.Println(styles.Boldf("Space CLI version already %s, no upgrade required", styles.Code(upgradingTo)))
		return nil
	}

	switch runtime.GOOS {
	case "linux", "darwin":
		return upgradeUnix()
	case "windows":
		return upgradeWin()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func upgradeUnix() error {
	curlCmd := exec.Command("curl", "-fsSL", "https://get.deta.dev/space-cli.sh")
	msg := "Upgrading Space CLI"
	curlOutput, err := curlCmd.CombinedOutput()
	if err != nil {
		logger.Println(string(curlOutput))
		return err
	}

	co := string(curlOutput)
	shCmd := exec.Command("sh", "-c", co)
	if versionFlag != "" {
		if !strings.HasPrefix(versionFlag, "v") {
			versionFlag = fmt.Sprintf("v%s", versionFlag)
		}
		msg = fmt.Sprintf("%s to version %s", msg, styles.Code(versionFlag))
		shCmd = exec.Command("sh", "-c", co, "upgrade", versionFlag)
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

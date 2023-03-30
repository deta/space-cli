//go:build windows
// +build windows

package version

import (
	"fmt"
	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/pkg/components/styles"
	"os/exec"
)

func upgradeWin(version string) error {
	msg := "Upgrading Space CLI"
	cmd := "iwr https://get.deta.dev/space-cli.ps1 -useb | iex"

	if version != "" {
		msg = fmt.Sprintf("%s to version %s", msg, styles.Code(version))
		cmd = fmt.Sprintf(`$v="%s"; %s`, version, cmd)
	}
	shared.Logger.Printf("%s...\n", msg)

	pshellCmd := exec.Command("powershell", cmd)

	stdout, err := pshellCmd.CombinedOutput()
	fmt.Println(string(stdout))
	if err != nil {
		return err
	}

	return nil
}

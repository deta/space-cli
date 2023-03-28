//go:build !windows
// +build !windows

package version

func upgradeWin(version string) error {
	return nil
}

package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/styles"
	"github.com/spf13/cobra"
)

type PreRunFunc func(cmd *cobra.Command, args []string) error

func CheckAll(funcs ...PreRunFunc) PreRunFunc {
	return func(cmd *cobra.Command, args []string) error {
		for _, f := range funcs {
			if f == nil {
				continue
			}
			if err := f(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

func CheckExists(flagName ...string) PreRunFunc {
	return func(cmd *cobra.Command, args []string) error {
		for _, flagName := range flagName {
			dir, _ := cmd.Flags().GetString(flagName)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return fmt.Errorf("directory %s does not exist", dir)
			}
		}
		return nil
	}
}

func CheckProjectInitialized(dirFlag string) PreRunFunc {
	return CheckAll(CheckExists(dirFlag), func(cmd *cobra.Command, args []string) error {
		if os.Getenv(runtime.SpaceProjectIDEnv) != "" {
			return nil
		}

		dir, _ := cmd.Flags().GetString(dirFlag)

		if _, err := os.Stat(filepath.Join(dir, ".space", "meta")); os.IsNotExist(err) {
			return errors.New("project is not initialized. run `space new` to initialize a new project or `space link` to associate an existing project.")
		}

		return nil
	})
}

func CheckNotEmpty(flagNames ...string) PreRunFunc {
	return func(cmd *cobra.Command, args []string) error {
		for _, flagName := range flagNames {
			if cmd.Flags().Changed(flagName) {
				value, _ := cmd.Flags().GetString(flagName)
				if strings.Trim(value, " ") == "" {
					return fmt.Errorf("%s cannot be empty", flagName)
				}
			}
		}
		return nil
	}
}

func isPrerelease(version string) bool {
	return len(strings.Split(version, "-")) > 1
}

func CheckLatestVersion(cmd *cobra.Command, args []string) error {
	if isPrerelease(SpaceVersion) {
		return nil
	}

	latestVersion, lastCheck, err := runtime.GetLatestCachedVersion()
	if err != nil || time.Since(lastCheck) > 69*time.Minute {
		Logger.Println("\nChecking for new Space CLI version...")
		version, err := api.GetLatestCliVersion()
		if err != nil {
			Logger.Println("Failed to check for new Space CLI version")
			return nil
		}

		runtime.CacheLatestVersion(version)
		latestVersion = version
	}

	if SpaceVersion != latestVersion {
		Logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}

	return nil
}

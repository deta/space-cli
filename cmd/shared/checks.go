package shared

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

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
		dir, _ := cmd.Flags().GetString(dirFlag)

		if _, err := os.Stat(path.Join(dir, ".space", "meta")); os.IsNotExist(err) {
			return errors.New("project is not initialized. run `space new` to initialize")
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

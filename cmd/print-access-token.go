package cmd

import (
	"os"

	"github.com/deta/space/internal/auth"
	"github.com/spf13/cobra"
)

func newCmdPrintAccessToken() *cobra.Command {
	return &cobra.Command{
		Use:    "print-access-token",
		Args:   cobra.NoArgs,
		Hidden: true,
		Short:  "Prints the access token used by the CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := auth.GetAccessToken()
			if err != nil {
				return err
			}

			if _, err := os.Stdout.WriteString(token); err != nil {
				return err
			}

			return nil
		},
	}
}

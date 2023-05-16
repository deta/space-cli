package cmd

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/deta/space/cmd/shared"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func newCmdAPI() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "api",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		Short:  "Makes an authenticated HTTP request to the Space API and prints the response.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var body []byte
			if !isatty.IsTerminal(os.Stdin.Fd()) {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				body = b
			}

			var method string
			if cmd.Flags().Changed("method") {
				method, _ := cmd.Flags().GetString("method")
				if strings.ToUpper(method) == "GET" && body != nil {
					return errors.New("cannot send body with GET request")
				}
			} else if body != nil {
				method = "POST"
			} else {
				method = "GET"
			}

			path := args[0]
			switch strings.ToUpper(method) {
			case "GET":
				res, err := shared.Client.Get(path)
				if err != nil {
					return err
				}
				os.Stdout.Write(res)
				return nil
			case "POST":
				res, err := shared.Client.Get(path)
				if err != nil {
					return err
				}
				os.Stdout.Write(res)
				return nil
			default:
				return errors.New("invalid method")
			}
		},
	}

	cmd.Flags().StringP("method", "X", "", "HTTP method")
	return cmd
}

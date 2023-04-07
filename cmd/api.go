package cmd

import (
	"io"
	"os"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/api"
	"github.com/spf13/cobra"
)

func newCmdAPI() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "api",
		Short:  "Make an authenticated API request to the Deta Space API",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			method, _ := cmd.Flags().GetString("method")
			var res *api.RequestOutput
			switch method {
			case "GET", "get":
				o, err := shared.Client.Get(args[0])
				if err != nil {
					shared.Logger.Fatal(err)
				}

				res = o
			case "POST", "post":
				var body []byte
				if !shared.IsInputInteractive() {
					b, err := io.ReadAll(os.Stdin)
					if err != nil {
						shared.Logger.Fatal(err)
					}

					body = b
				}
				o, err := shared.Client.Post(args[0], body)
				if err != nil {
					shared.Logger.Fatal(err)
				}

				res = o
			default:
				shared.Logger.Fatal("invalid method")
			}

			os.Stdout.Write(res.Body)
		},
	}

	cmd.Flags().StringP("method", "X", "GET", "HTTP method to use")

	return cmd
}

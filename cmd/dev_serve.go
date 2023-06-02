package cmd

import (
	"fmt"
	"net/http"

	"github.com/deta/space/cmd/utils"
	"github.com/spf13/cobra"
)

func newCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "serve",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		Short:  "Serve static files",
		Run: func(cmd *cobra.Command, args []string) {
			fs := http.FileServer(http.Dir(args[0]))
			http.Handle("/", fs)

			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")

			address := fmt.Sprintf("%s:%d", host, port)
			utils.Logger.Printf("Serving %s on %s", args[0], address)
			http.ListenAndServe(address, nil)
		},
	}

	cmd.Flags().IntP("port", "p", 8080, "port to serve on")
	cmd.Flags().StringP("host", "H", "localhost", "host to serve on")

	return cmd
}

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/deta/space/cmd/shared"
	"github.com/itchyny/gojq"
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
				method, _ = cmd.Flags().GetString("method")
				if strings.ToUpper(method) == "GET" && body != nil {
					return errors.New("cannot send body with GET request")
				}
			} else if body != nil {
				method = "POST"
			} else {
				method = "GET"
			}

			path := args[0]
			var res []byte
			switch strings.ToUpper(method) {
			case "GET":
				r, err := shared.Client.Get(path)
				if err != nil {
					return err
				}

				res = r
			case "POST":
				r, err := shared.Client.Post(path, body)
				if err != nil {
					return err
				}

				res = r
			case "DELETE":
				r, err := shared.Client.Delete(path, body)
				if err != nil {
					return err
				}

				res = r
			default:
				return errors.New("invalid method")
			}

			if !cmd.Flags().Changed("jq") {
				os.Stdout.Write(res)
				return nil
			}

			jq, _ := cmd.Flags().GetString("jq")
			query, err := gojq.Parse(jq)
			if err != nil {
				return fmt.Errorf("invalid jq query: %s", err)
			}

			var v any
			if err := json.Unmarshal(res, &v); err != nil {
				return err
			}

			encoder := json.NewEncoder(os.Stdout)
			if isatty.IsTerminal(os.Stdout.Fd()) {
				encoder.SetIndent("", "  ")
			}

			iter := query.Run(v)
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := v.(error); ok {
					log.Fatalln(err)
				}
				if err := encoder.Encode(v); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringP("method", "X", "", "HTTP method")
	cmd.Flags().String("jq", "", "jq filter")
	return cmd
}

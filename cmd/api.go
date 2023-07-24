package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/auth"
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
			method, _ := cmd.Flags().GetString("method")
			method = strings.ToUpper(method)

			var body []byte
			if cmd.Flags().Changed("body") {
				data, _ := cmd.Flags().GetString("data")
				if data != "" {
					body = []byte(data)
				}
			}

			if method == "GET" && len(body) > 0 {
				return fmt.Errorf("cannot use GET method with body")
			}

			url := args[0]
			if !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "https") {
				if !strings.HasPrefix(url, "/") {
					url = "/" + url
				}

				if !strings.HasPrefix(url, "/v0") {
					url = "/v0" + url
				}

				url = fmt.Sprintf("https://deta.space/api%s", url)
			}

			req, err := http.NewRequest(method, url, bytes.NewReader(body))
			if err != nil {
				return err
			}

			accessToken, err := auth.GetAccessToken()
			if err != nil {
				return err
			}

			if err := prepareRequest(accessToken, req); err != nil {
				return err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			res, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
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

	cmd.Flags().StringP("method", "X", "GET", "HTTP method")
	cmd.Flags().StringP("data", "d", "", "HTTP request body")
	cmd.Flags().String("jq", "", "jq filter")
	return cmd
}

func prepareRequest(accessToken string, req *http.Request) error {
	if req.URL.Hostname() == "deta.space" {
		return utils.Client.AuthenticateRequest(accessToken, req)
	} else if req.URL.Hostname() == "database.deta.sh" || req.URL.Hostname() == "drive.deta.sh" {
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) < 3 {
			return fmt.Errorf("invalid path: %s", req.URL.Path)
		}
		projectID := parts[2]
		dataKey, err := utils.GenerateDataKeyIfNotExists(projectID)
		if err != nil {
			return fmt.Errorf("failed to generate data key: %w", err)
		}

		req.Header.Set("X-Api-Key", dataKey)
		return nil
	} else {
		hostname := req.URL.Hostname()

		apiKey, err := utils.GenerateApiKeyIfNotExists(accessToken, hostname)
		if err != nil {
			return fmt.Errorf("failed to generate api key: %w", err)
		}

		req.Header.Set("X-Space-App-Key", apiKey)
		return nil
	}
}

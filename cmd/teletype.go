package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deta/space/cmd/shared"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

type Action struct {
	InstanceID    string `json:"instance_id"`
	InstanceAlias string `json:"instance_alias"`
	AppName       string `json:"app_name"`
	ActionName    string `json:"action_name"`
	ActionID      string `json:"action_id"`
}

func newTeletypeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tty {<instance-id> | <instance-alias>} <action-id>",
		Short: "Trigger a app action. Input is read from stdin.",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			body, err := shared.Client.Get("/v0/actions")
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var actions []Action
			if err = json.Unmarshal(body, &actions); err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			instance2actions := make(map[string][]Action)
			for _, action := range actions {
				instance2actions[action.InstanceAlias] = append(instance2actions[action.InstanceAlias], action)
			}

			if len(args) == 0 {
				args := make([]string, 0)
				for instanceAlias, actions := range instance2actions {
					if strings.HasPrefix(instanceAlias, toComplete) {
						appName := actions[0].AppName
						args = append(args, fmt.Sprintf("%s\t%s", instanceAlias, appName))
					}
				}

				return args, cobra.ShellCompDirectiveNoFileComp
			} else {
				instanceAlias := args[0]
				actions, ok := instance2actions[instanceAlias]
				if !ok {
					return nil, cobra.ShellCompDirectiveError
				}

				args := make([]string, 0)
				for _, action := range actions {
					if strings.HasPrefix(action.ActionID, toComplete) {
						args = append(args, fmt.Sprintf("%s\t%s", action.ActionID, action.ActionName))
					}
				}

				return args, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var input string
			if !isatty.IsTerminal(os.Stdin.Fd()) {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}

				input = string(b)
			}

			body, err := shared.Client.Get("/v0/actions")
			if err != nil {
				return err
			}

			var actions []Action
			if err = json.Unmarshal(body, &actions); err != nil {
				return err
			}

			for _, action := range actions {
				if action.InstanceAlias != args[0] && action.InstanceID != args[0] {
					continue
				}

				if action.ActionID != args[1] {
					continue
				}

				path := fmt.Sprintf("/v0/actions/%s/%s/invoke", action.InstanceID, action.ActionID)
				body, err := shared.Client.Post(path, []byte(input))
				if err != nil {
					return err
				}

				if _, err := os.Stdout.Write(body); err != nil {
					return err
				}

				return nil
			}

			return nil
		},
	}

	return cmd
}

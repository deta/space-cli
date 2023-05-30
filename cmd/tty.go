package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/deta/space/cmd/shared"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

type Action struct {
	InstanceID    string      `json:"instance_id"`
	InstanceAlias string      `json:"instance_alias"`
	AppName       string      `json:"app_name"`
	Name          string      `json:"name"`
	Title         string      `json:"title"`
	Input         ActionInput `json:"input"`
}

type ActionInput []struct {
	Name     string    `json:"name"`
	Type     InputType `json:"type"`
	Optional bool      `json:"optional"`
}

type InputType string

var (
	InputTypeString InputType = "string"
	InputTypeBool   InputType = "bool"
)

func parseFlags(args []string, action Action) (map[string]any, error) {
	flags := make(map[string]any)

	for len(args) > 0 {
		arg := args[0]
		args = args[1:]
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("invalid flag %s", arg)
		}

		flag := strings.TrimPrefix(arg, "--")
		parts := strings.Split(flag, "=")
		var inputType InputType
		for _, input := range action.Input {
			if input.Name != parts[0] {
				continue
			}

			inputType = input.Type
		}

		switch inputType {
		case InputTypeString:
			if len(parts) == 2 {
				flags[parts[0]] = parts[1]
			} else {
				if len(args) == 0 {
					return nil, fmt.Errorf("invalid flag %s", arg)
				}

				value := args[0]
				args = args[1:]
				if strings.HasPrefix(value, "--") {
					return nil, fmt.Errorf("invalid flag %s", arg)
				}

				flags[parts[0]] = args[0]
			}
		case InputTypeBool:
			flags[flag] = true
		default:
			return nil, fmt.Errorf("invalid flag %s", arg)
		}

	}
	return flags, nil
}

func newCmdTTY() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "tty {<instance-id> | <instance-alias>} <action-id>",
		Short:              "Trigger a app action. Input is read from stdin.",
		Args:               cobra.MinimumNArgs(2),
		DisableFlagParsing: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			body, err := shared.Client.Get("/v0/actions")
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var actions []Action
			if err = json.Unmarshal(body, &actions); err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			if shared.IsPortActive(shared.DevPort) {
				res, err := http.Get(fmt.Sprintf("http://localhost:%d/__space/actions", shared.DevPort))
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}
				defer res.Body.Close()

				var devActions []Action
				if err = json.NewDecoder(res.Body).Decode(&devActions); err != nil {
					return nil, cobra.ShellCompDirectiveError
				}

				actions = append(actions, devActions...)
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
			} else if len(args) == 1 {
				instanceAlias := args[0]
				actions, ok := instance2actions[instanceAlias]
				if !ok {
					return nil, cobra.ShellCompDirectiveError
				}

				args := make([]string, 0)
				for _, action := range actions {
					if strings.HasPrefix(action.Title, toComplete) {
						args = append(args, fmt.Sprintf("%s\t%s", action.Name, action.Title))
					}
				}

				return args, cobra.ShellCompDirectiveNoFileComp
			} else {
				instanceAlias := args[0]
				actions, ok := instance2actions[instanceAlias]
				if !ok {
					return nil, cobra.ShellCompDirectiveError
				}

				var action *Action
				for _, a := range actions {
					if a.Name == args[1] {
						action = &a
						break
					}
				}

				if action == nil {
					return nil, cobra.ShellCompDirectiveError
				}

				args := make([]string, 0)
				for _, input := range action.Input {
					if strings.HasPrefix("--"+input.Name, toComplete) {
						args = append(args, fmt.Sprintf("--%s\t%s", input.Name, input.Name))
					}
				}

				return args, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var input map[string]any
			if !isatty.IsTerminal(os.Stdin.Fd()) {
				bs, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}

				if err := json.Unmarshal(bs, &input); err != nil {
					return err
				}
			}

			body, err := shared.Client.Get("/v0/actions")
			if err != nil {
				return err
			}

			var actions []Action
			if err = json.Unmarshal(body, &actions); err != nil {
				return err
			}

			if shared.IsPortActive(shared.DevPort) {
				res, err := http.Get(fmt.Sprintf("http://localhost:%d/__space/actions", shared.DevPort))
				if err != nil {
					return err
				}
				defer res.Body.Close()

				var devActions []Action
				if err = json.NewDecoder(res.Body).Decode(&devActions); err != nil {
					return err
				}

				actions = append(actions, devActions...)
			}

			for _, action := range actions {
				if action.InstanceAlias != args[0] && action.InstanceID != args[0] {
					continue
				}

				if action.Title != args[1] {
					continue
				}

				flags, err := parseFlags(args[2:], action)
				if err != nil {
					return err
				}

				for k, v := range flags {
					input[k] = v
				}

				path := fmt.Sprintf("/v0/actions/%s/%s/invoke", action.InstanceID, action.Title)
				payload, err := json.Marshal(input)
				if err != nil {
					return err
				}

				body, err := shared.Client.Post(path, payload)
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

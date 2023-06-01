package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/deta/space/cmd/shared"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

type ActionResponse struct {
	Actions []Action `json:"actions"`
}

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

func newCmdTTY() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tty <instance-id> <action-name>",
		Short: "Trigger a app action",
		Long:  `Trigger a app action.If the action requires input, it will be prompted for. You can also pipe the input to the command, or pass it as a flag.`,
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			body, err := shared.Client.Get("/v0/actions")
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var actionResponse ActionResponse
			if err = json.Unmarshal(body, &actionResponse); err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			actions := actionResponse.Actions

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
				instance2actions[action.InstanceID] = append(instance2actions[action.InstanceID], action)
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
			} else if len(args) == 1 || len(args) == 2 {
				instanceAlias := args[0]
				actions, ok := instance2actions[instanceAlias]
				if !ok {
					return nil, cobra.ShellCompDirectiveError
				}

				args := make([]string, 0)
				for _, action := range actions {
					if strings.HasPrefix(action.Name, toComplete) {
						args = append(args, fmt.Sprintf("%s\t%s", action.Name, action.Title))
					}
				}

				return args, cobra.ShellCompDirectiveNoFileComp
			} else {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var action Action
			if args[0] == "dev" {
				if !shared.IsPortActive(shared.DevPort) {
					return fmt.Errorf("dev server is not running")
				}
				res, err := http.Get(fmt.Sprintf("http://localhost:%d/__space/actions", shared.DevPort))
				if err != nil {
					return err
				}
				defer res.Body.Close()

				if err = json.NewDecoder(res.Body).Decode(&action); err != nil {
					return err
				}
			} else {
				body, err := shared.Client.Get(fmt.Sprintf("/v0/actions/%s/%s", args[0], args[1]))
				if err != nil {
					return err
				}

				if err = json.Unmarshal(body, &action); err != nil {
					return err
				}
			}

			var stdinParams map[string]any
			if !isatty.IsTerminal(os.Stdin.Fd()) {
				bs, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}

				if err := json.Unmarshal(bs, &stdinParams); err != nil {
					return err
				}
			}

			inputParams := make(map[string]any)
			inputFlag, _ := cmd.Flags().GetStringArray("input")
			for _, input := range inputFlag {
				parts := strings.Split(input, "=")
				if len(parts) != 2 {
					return fmt.Errorf("invalid input flag: %s", input)
				}

				inputParams[parts[0]] = parts[1]
			}

			payload := make(map[string]any)
			for _, input := range action.Input {
				if param, ok := inputParams[input.Name]; ok {
					payload[input.Name] = param
					continue
				}

				if param, ok := stdinParams[input.Name]; ok {
					payload[input.Name] = param
					continue
				}

				if input.Optional {
					continue
				}

				switch input.Type {
				case "string":
					var res string
					if err := survey.AskOne(&survey.Input{Message: fmt.Sprintf("%s:", input.Name)}, &res, nil); err != nil {
						return err
					}

					payload[input.Name] = res
				case "number":
					var res int
					if err := survey.AskOne(
						&survey.Input{Message: fmt.Sprintf("%s:", input.Name)},
						&res,
						survey.WithValidator(func(ans interface{}) error {
							if _, err := strconv.Atoi(ans.(string)); err != nil {
								return fmt.Errorf("invalid number")
							}
							return nil
						},
						)); err != nil {
						return err
					}

					payload[input.Name] = res
				case "boolean":
					var res bool
					if err := survey.AskOne(&survey.Confirm{Message: fmt.Sprintf("%s:", input.Name)}, &res); err != nil {
						return err
					}

					payload[input.Name] = res
				default:
					return fmt.Errorf("unknown input type: %s", input.Type)
				}
			}

			body, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/v0/actions/%s/%s", action.InstanceID, action.Name)
			if action.InstanceID == "dev" {
				path = fmt.Sprintf("http://localhost:%d/__space/actions/%s", shared.DevPort, action.Title)
			}

			res, err := shared.Client.Post(path, body)
			if err != nil {
				return err
			}

			if _, err := os.Stdout.Write(res); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringArrayP("input", "i", nil, "Input parameters")

	return cmd
}
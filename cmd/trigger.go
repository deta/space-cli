package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/deta/space/cmd/utils"
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

func newCmdTrigger() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "trigger <instance-alias> <action-name>",
		Short:  "Trigger a app action",
		Long:   `Trigger a app action.If the action requires input, it will be prompted for. You can also pipe the input to the command, or pass it as a flag.`,
		Hidden: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			body, err := utils.Client.Get("/v0/actions")
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var actionResponse ActionResponse
			if err = json.Unmarshal(body, &actionResponse); err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			actions := actionResponse.Actions
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
			res, err := utils.Client.Get("/v0/actions?per_page=1000")
			if err != nil {
				return err
			}

			var actionResponse ActionResponse
			if err = json.Unmarshal(res, &actionResponse); err != nil {
				return err
			}

			alias2actions := make(map[string][]Action)
			for _, action := range actionResponse.Actions {
				if len(args) > 0 && !strings.HasPrefix(action.InstanceAlias, args[0]) {
					continue
				}
				alias2actions[action.InstanceAlias] = append(alias2actions[action.InstanceAlias], action)
			}

			var actions []Action
			if len(alias2actions) == 0 {
				return fmt.Errorf("no instances found")
			} else if len(alias2actions) == 1 && len(args) > 0 {
				for _, items := range alias2actions {
					actions = append(actions, items...)
				}
			} else {
				instanceAliases := make([]string, 0)
				for alias := range alias2actions {
					instanceAliases = append(instanceAliases, alias)
				}

				var response string
				survey.AskOne(&survey.Select{
					Message: "Select an instance:",
					Options: instanceAliases,
					Description: func(value string, index int) string {
						actions := alias2actions[value]
						return actions[0].AppName
					},
				}, &response)

				actions = alias2actions[response]
			}

			var action *Action
			if len(args) > 1 {
				for _, a := range actions {
					if a.Name == args[1] {
						action = &a
						break
					}
				}

				if action == nil {
					return fmt.Errorf("action %s not found", args[1])
				}
			} else {
				options := make([]string, 0)
				for _, a := range actions {
					options = append(options, a.Name)
				}

				var response string
				if err := survey.AskOne(
					&survey.Select{
						Message: "Select an action:",
						Options: options,
						Description: func(value string, index int) string {
							return actions[index].Title
						},
					},
					&response,
				); err != nil {
					return err
				}

				for _, a := range actions {
					if a.Name == response {
						action = &a
						break
					}
				}

				if action == nil {
					return fmt.Errorf("action %s not found", response)
				}
			}

			payload, err := extractInput(cmd, *action)
			if err != nil {
				return err
			}

			body, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/v0/actions/%s/%s", action.InstanceID, action.Name)
			actionRes, err := utils.Client.Post(path, body)
			if err != nil {
				return err
			}

			os.Stdout.Write(actionRes)

			return nil
		},
	}

	cmd.Flags().StringArrayP("input", "i", nil, "Input parameters")

	return cmd
}

func extractInput(cmd *cobra.Command, action Action) (map[string]any, error) {
	params := make(map[string]any)
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		var stdinParams map[string]any
		bs, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(bs, &stdinParams); err != nil {
			return nil, err
		}

		for k, v := range stdinParams {
			params[k] = v
		}
	}

	if cmd.Flags().Changed("input") {
		inputFlag, _ := cmd.Flags().GetStringArray("input")
		for _, input := range inputFlag {
			parts := strings.Split(input, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid input flag: %s", input)
			}

			params[parts[0]] = parts[1]
		}
	}

	payload := make(map[string]any)
	for _, input := range action.Input {
		if param, ok := params[input.Name]; ok {
			payload[input.Name] = param
			continue
		}

		if input.Optional {
			continue
		}

		switch input.Type {
		case "string":
			var res string
			prompt := &survey.Input{Message: fmt.Sprintf("Input %s:", input.Name)}
			if err := survey.AskOne(prompt, &res, nil); err != nil {
				return nil, err
			}

			payload[input.Name] = res
		case "number":
			var res int
			prompt := &survey.Input{Message: fmt.Sprintf("Input %s:", input.Name)}
			validator := func(ans interface{}) error {
				if _, err := strconv.Atoi(ans.(string)); err != nil {
					return fmt.Errorf("invalid number")
				}
				return nil
			}
			if err := survey.AskOne(prompt, &res, survey.WithValidator(validator)); err != nil {
				return nil, err
			}

			payload[input.Name] = res
		case "boolean":
			var res bool
			prompt := &survey.Confirm{Message: fmt.Sprintf("Input %s:", input.Name)}
			if err := survey.AskOne(prompt, &res); err != nil {
				return nil, err
			}

			payload[input.Name] = res
		default:
			return nil, fmt.Errorf("unknown input type: %s", input.Type)
		}
	}

	return payload, nil
}

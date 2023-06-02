package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/deta/space/shared"
	"github.com/spf13/cobra"
)

func newCmdDevTrigger() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <action>",
		Short: "Trigger a micro action",
		Long: `Manually trigger an action.
Make sure that the corresponding micro is running before triggering the action.`,
		Aliases:  []string{"t"},
		Args:     cobra.MaximumNArgs(1),
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			experimental, _ := cmd.Flags().GetBool("experimental")
			if !experimental {
				projectDir, _ := cmd.Flags().GetString("dir")

				if len(args) == 0 {
					return errors.New("action name is required")
				}

				if err := triggerScheduleAction(projectDir, args[0]); err != nil {
					return err
				}

				return nil
			}

			var action Action
			if len(args) > 0 {
				actionRes, err := http.Get(fmt.Sprintf("http://localhost:%d/__space/actions/%s", utils.DevPort, args[0]))
				if err != nil {
					return err
				}
				defer actionRes.Body.Close()

				if err := json.NewDecoder(actionRes.Body).Decode(&action); err != nil {
					return err
				}

			} else {
				if !utils.IsPortActive(utils.DevPort) {
					utils.Logger.Printf("%s No action specified and no micro is running", emoji.X)
				}

				res, err := http.Get(fmt.Sprintf("http://localhost:%d/__space/actions", utils.DevPort))
				if err != nil {
					return err
				}
				defer res.Body.Close()

				var actions []Action
				if err := json.NewDecoder(res.Body).Decode(&actions); err != nil {
					return err
				}

				options := make([]string, len(actions))
				for i, action := range actions {
					options[i] = action.Name
				}

				var response string
				if err := survey.AskOne(&survey.Select{
					Message: "Select an action to trigger",
					Options: options,
					Description: func(value string, index int) string {
						return actions[index].Title
					},
				}, &response); err != nil {
					return err
				}

				for _, a := range actions {
					if a.Name == response {
						action = a
						break
					}
				}

				if action.Name == "" {
					return fmt.Errorf("action %s not found", response)
				}
			}

			params, err := extractInput(cmd, action)
			if err != nil {
				return err
			}

			payload, err := json.Marshal(params)
			if err != nil {
				return err
			}

			actionResponse, err := http.Post(fmt.Sprintf("http://localhost:%d/__space/actions/%s", utils.DevPort, action.Name), "application/json", bytes.NewReader(payload))
			if err != nil {
				return err
			}
			defer actionResponse.Body.Close()

			utils.Logger.Println("\n┌ Action Response:")
			io.Copy(os.Stdout, actionResponse.Body)

			return nil
		},
	}

	cmd.Flags().StringArrayP("input", "i", []string{}, "action input")
	cmd.Flags().Bool("experimental", false, "enable experimental features")
	cmd.Flags().String("id", "", "project id")

	return cmd
}

func triggerScheduleAction(projectDir string, actionID string) (err error) {
	spacefile, err := spacefile.LoadSpacefile(projectDir)
	if err != nil {
		utils.Logger.Printf("%s failed to parse Spacefile: %s", emoji.X, err.Error())
	}
	routeDir := filepath.Join(projectDir, ".space", "micros")

	for _, micro := range spacefile.Micros {
		for _, action := range micro.Actions {
			if action.ID != actionID {
				continue
			}

			utils.Logger.Printf("\n%s Checking if micro %s is running...\n", emoji.Eyes, styles.Green(micro.Name))
			port, err := getMicroPort(micro, routeDir)
			if err != nil {
				upCommand := fmt.Sprintf("space dev up %s", micro.Name)
				utils.Logger.Printf("%smicro %s is not running, to start it run:", emoji.X, styles.Green(micro.Name))
				utils.Logger.Printf("L %s", styles.Blue(upCommand))
				return err
			}

			utils.Logger.Printf("%s Micro %s is running", styles.Green("✔️"), styles.Green(micro.Name))

			body, err := json.Marshal(shared.ActionRequest{
				Event: shared.ActionEvent{
					ID:      actionID,
					Trigger: "schedule",
				},
			})
			if err != nil {
				return err
			}

			actionEndpoint := fmt.Sprintf("http://localhost:%d/%s", port, actionEndpoint)
			utils.Logger.Printf("\nTriggering action %s", styles.Green(actionID))
			utils.Logger.Printf("L POST %s", styles.Blue(actionEndpoint))

			res, err := http.Post(actionEndpoint, "application/json", bytes.NewReader(body))
			if err != nil {
				utils.Logger.Printf("\n%s failed to trigger action: %s", emoji.X, err.Error())
				return err
			}
			defer res.Body.Close()

			utils.Logger.Println("\n┌ Action Response:")

			utils.Logger.Printf("\n%s", res.Status)

			utils.Logger.Println()
			io.Copy(os.Stdout, res.Body)

			if res.StatusCode >= 400 {
				utils.Logger.Printf("\n\nL %s", styles.Error("failed to trigger action"))
				return err
			}
			utils.Logger.Printf("\n\nL Action triggered successfully!")
			return nil
		}
	}

	utils.Logger.Printf("\n%saction `%s` not found", emoji.X, actionID)
	return fmt.Errorf("\n%saction `%s` not found", emoji.X, actionID)
}

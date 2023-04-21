package dev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	types "github.com/deta/space/shared"
	"github.com/spf13/cobra"
)

func newCmdDevTrigger() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <action>",
		Short: "Trigger a micro action",
		Long: `Manually trigger an action.
Make sure that the corresponding micro is running before triggering the action.`,
		Aliases:  []string{"t"},
		Args:     cobra.ExactArgs(1),
		PreRunE:  shared.CheckProjectInitialized("dir"),
		PostRunE: shared.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")

			if err := devTrigger(projectDir, args[0]); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP("dir", "d", ".", "project directory")

	return cmd
}

func devTrigger(projectDir string, actionID string) (err error) {
	spacefile, err := spacefile.LoadSpacefile(projectDir)
	if err != nil {
		shared.Logger.Printf("%s failed to parse Spacefile: %s", emoji.X, err.Error())
	}
	routeDir := filepath.Join(projectDir, ".space", "micros")

	for _, micro := range spacefile.Micros {
		for _, action := range micro.Actions {
			if action.ID != actionID {
				continue
			}

			shared.Logger.Printf("\n%s Checking if micro %s is running...\n", emoji.Eyes, styles.Green(micro.Name))
			port, err := getMicroPort(micro, routeDir)
			if err != nil {
				upCommand := fmt.Sprintf("space dev up %s", micro.Name)
				shared.Logger.Printf("%smicro %s is not running, to start it run:", emoji.X, styles.Green(micro.Name))
				shared.Logger.Printf("L %s", styles.Blue(upCommand))
				return err
			}

			shared.Logger.Printf("%s Micro %s is running", styles.Green("✔️"), styles.Green(micro.Name))

			body, err := json.Marshal(types.ActionRequest{
				Event: types.ActionEvent{
					ID:      actionID,
					Trigger: "schedule",
				},
			})
			if err != nil {
				return err
			}

			actionEndpoint := fmt.Sprintf("http://localhost:%d/%s", port, actionEndpoint)
			shared.Logger.Printf("\nTriggering action %s", styles.Green(actionID))
			shared.Logger.Printf("L POST %s", styles.Blue(actionEndpoint))

			res, err := http.Post(actionEndpoint, "application/json", bytes.NewReader(body))
			if err != nil {
				shared.Logger.Printf("\n%s failed to trigger action: %s", emoji.X, err.Error())
				return err
			}
			defer res.Body.Close()

			shared.Logger.Println("\n┌ Action Response:")

			shared.Logger.Printf("\n%s", res.Status)

			shared.Logger.Println()
			io.Copy(os.Stdout, res.Body)

			if res.StatusCode >= 400 {
				shared.Logger.Printf("\n\nL %s", styles.Error("failed to trigger action"))
				return err
			}
			shared.Logger.Printf("\n\nL Action triggered successfully!")
			return nil
		}
	}

	shared.Logger.Printf("\n%saction `%s` not found", emoji.X, actionID)
	return fmt.Errorf("\n%saction `%s` not found", emoji.X, actionID)
}

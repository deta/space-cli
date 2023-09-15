package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/runtime"
	"github.com/spf13/cobra"
)

func newCmdExec() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Run a command in the context of your project",
		Long: `Run a command in the context of your project.

The data key will be automatically injected into the command's environment.`,
		Args:     cobra.MinimumNArgs(1),
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			projectID, _ := cmd.Flags().GetString("project")
			if !cmd.Flags().Changed("project") {
				cwd, _ := os.Getwd()
				projectID, err = runtime.GetProjectID(cwd)
				if err != nil {
					return fmt.Errorf("project id not provided and could not be inferred from current working directory")
				}
			}

			if err := execRun(projectID, args); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("project", "", "id of project to exec the command in")

	return cmd
}

func execRun(projectID string, args []string) error {
	var err error

	projectKey, err := utils.GenerateDataKeyIfNotExists(projectID)
	if err != nil {
		return fmt.Errorf("failed to generate data key: %w", err)
	}

	name := args[0]
	var extraArgs []string
	if len(args) > 1 {
		extraArgs = args[1:]
	}

	command := exec.Command(name, extraArgs...)
	command.Env = os.Environ()
	command.Env = append(command.Env, "DETA_PROJECT_KEY="+projectKey)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	return command.Run()
}

package cmd

import (
	"os"
	"os/exec"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/spf13/cobra"
)

func newCmdExec() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Run a command in the context of your project",
		Long: `Run a command in the context of your project.

The data key will be automatically injected into the command's environment.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			projectID, _ := cmd.Flags().GetString("project")
			if !cmd.Flags().Changed("project") {
				cwd, _ := os.Getwd()
				projectID, err = runtime.GetProjectID(cwd)
				if err != nil {
					shared.Logger.Printf("project id not provided and could not be inferred from current working directory")
					os.Exit(1)
				}
			}

			if err := execRun(projectID, args); err != nil {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().String("project", "", "id of project to exec the command in")
	cmd.MarkFlagRequired("project")

	return cmd
}

func execRun(projectID string, args []string) error {
	var err error

	projectKey, err := shared.GenerateDataKeyIfNotExists(projectID)
	if err != nil {
		shared.Logger.Printf("%sError generating data key: %s\n", emoji.ErrorExclamation, err.Error())
		return err
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

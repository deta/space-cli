package cmd

import (
	"os"
	"os/exec"

	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/spf13/cobra"
)

var (
	execCmd = &cobra.Command{
		Use:   "exec",
		Short: "executes a command in space context",
		Args:  cobra.MinimumNArgs(1),
		Run:   execRun,
	}
)

func init() {
	execCmd.Flags().String("project", "", "id of project to exec the command in")
	execCmd.MarkFlagRequired("project")
	rootCmd.AddCommand(execCmd)
}

func execRun(cmd *cobra.Command, args []string) {
	projectId, _ := cmd.Flags().GetString("project")
	projectKey, err := generateDataKeyIfNotExists(projectId)
	if err != nil {
		logger.Printf("%sError generating data key: %s\n", emoji.ErrorExclamation, err.Error())
		return
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

	command.Run()
}

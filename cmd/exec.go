package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"

	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/spf13/cobra"
)

var (
	execCmd = &cobra.Command{
		Use:    "exec",
		Short:  "executes a command in space context",
		Args:   cobra.MinimumNArgs(1),
		PreRun: execPreRun,
		Run:    execRun,
	}
)

func init() {
	execCmd.Flags().String("project", "", "id of project to exec the command in")
	execCmd.MarkFlagRequired("project")
	rootCmd.AddCommand(execCmd)
}

func execPreRun(cmd *cobra.Command, args []string) {
	if cmd.Flags().Changed("project") {
		return
	}

	// if the project flag is not set, try to find the project id in the current directory
	cwd, _ := os.Getwd()
	metaPath := path.Join(cwd, ".space", "meta")
	bytes, err := os.ReadFile(metaPath)
	if err != nil {
		logger.Printf("%sproject flag is required when not in a space directory.", emoji.X)
		os.Exit(1)
	}

	var meta runtime.ProjectMeta
	if err := json.Unmarshal(bytes, &meta); err != nil {
		logger.Printf("%sCould not parse project metadatas.", emoji.X)
		os.Exit(1)
	}

	cmd.Flags().Set("project", meta.ID)
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

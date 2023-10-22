package cmd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/pkg/components/styles"
	"github.com/spf13/cobra"
)

func newCmdGenerate() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "generate [flags]",
		Short:    "Create a Spacefile without a new project in Space",
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")
			blankProject, _ := cmd.Flags().GetBool("blank")
			projectName, _ := cmd.Flags().GetString("name")

			if !cmd.Flags().Changed("name") {
				abs, err := filepath.Abs(projectDir)
				if err != nil {
					utils.Logger.Printf("%sError getting absolute path of project directory: %s", styles.ErrorExclamation, err.Error())
					return err
				}

				name := filepath.Base(abs)
				projectName, err = selectProjectName(name)
				if err != nil {
					return err
				}
			}

			// Create spacefile if it doesn't exist
			spaceFilePath := filepath.Join(projectDir, "Spacefile")
			if _, err := os.Stat(spaceFilePath); errors.Is(err, os.ErrNotExist) {
				err := createSpacefile(projectDir, projectName, blankProject)
				if err != nil {
					utils.Logger.Printf("failed to create spacefile: %s", err)
					return err
				}
			}

			return nil
		},
		PreRunE: utils.CheckAll(
			utils.CheckExists("dir"),
			func(cmd *cobra.Command, args []string) error {
				if cmd.Flags().Changed("name") {
					name, _ := cmd.Flags().GetString("name")
					return validateProjectName(name)
				}

				return nil
			}),
	}

	cmd.Flags().StringP("name", "n", "", "project name")
	cmd.Flags().StringP("dir", "d", "./", "src of project to release")
	cmd.MarkFlagDirname("dir")
	cmd.Flags().BoolP("blank", "b", false, "create blank project")

	if !utils.IsOutputInteractive() {
		cmd.MarkFlagRequired("name")
	}

	return cmd
}

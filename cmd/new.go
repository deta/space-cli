package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/auth"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/confirm"
	"github.com/deta/space/pkg/components/styles"
	"github.com/deta/space/pkg/components/text"
	"github.com/deta/space/pkg/scanner"
	"github.com/spf13/cobra"
)

func newCmdNew() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "new [flags]",
		Short:    "Create new project",
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")
			blankProject, _ := cmd.Flags().GetBool("blank")
			projectName, _ := cmd.Flags().GetString("name")

			if !cmd.Flags().Changed("name") {
				abs, err := filepath.Abs(projectDir)
				if err != nil {
					return fmt.Errorf("failed to get absolute path of project directory: %w", err)
				}

				name := filepath.Base(abs)
				projectName, err = selectProjectName(name)
				if err != nil {
					return err
				}
			}

			if err := newProject(projectDir, projectName, blankProject); err != nil {
				return err
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

func validateProjectName(projectName string) error {
	if len(projectName) < 4 {
		return fmt.Errorf("project name must be at least 4 characters long")
	}

	if len(projectName) > 16 {
		return fmt.Errorf("project name must be at most 16 characters long")
	}

	return nil
}

func selectProjectName(placeholder string) (string, error) {
	promptInput := text.Input{
		Prompt:      "What is your project's name?",
		Placeholder: placeholder,
		Validator:   validateProjectName,
	}

	return text.Run(&promptInput)
}

func createProject(name string) (*runtime.ProjectMeta, error) {
	res, err := utils.Client.CreateProject(&api.CreateProjectRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	return &runtime.ProjectMeta{ID: res.ID, Name: res.Name, Alias: res.Alias}, nil
}

func createSpacefile(projectDir string, projectName string, blankProject bool) error {
	if blankProject {
		_, err := spacefile.CreateBlankSpacefile(projectDir)
		return err
	}

	autoDetectedMicros, err := scanner.Scan(projectDir)
	if err != nil {
		return fmt.Errorf("problem while trying to auto detect runtimes/frameworks for project %s: %s", projectName, err)
	}

	if len(autoDetectedMicros) == 0 {
		_, err := spacefile.CreateBlankSpacefile(projectDir)
		return err
	}

	for _, micro := range autoDetectedMicros {
		utils.Logger.Printf("\nMicro found in \"%s\"", styles.Code(micro.Src))
		utils.Logger.Printf("L engine: %s\n", styles.Blue(micro.Engine))
	}

	if !utils.IsOutputInteractive() {
		_, err = spacefile.CreateSpacefileWithMicros(projectDir, autoDetectedMicros)
		return err
	}

	utils.Logger.Println()
	if ok, err := confirm.Run(fmt.Sprintf("Do you want to setup \"%s\" with this configuration?", projectName)); err != nil {
		return err
	} else if !ok {
		_, err := spacefile.CreateBlankSpacefile(projectDir)
		return err
	}

	_, err = spacefile.CreateSpacefileWithMicros(projectDir, autoDetectedMicros)
	return err
}

func newProject(projectDir, projectName string, blankProject bool) error {
	// Create spacefile if it doesn't exist
	spaceFilePath := filepath.Join(projectDir, "Spacefile")
	if _, err := os.Stat(spaceFilePath); errors.Is(err, os.ErrNotExist) {
		err := createSpacefile(projectDir, projectName, blankProject)
		if err != nil {
			return fmt.Errorf("failed to create Spacefile, %w", err)
		}
	}

	// Create project
	meta, err := createProject(projectName)
	if err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			utils.Logger.Println(utils.LoginInfo())
			return err
		}
		return fmt.Errorf("failed to create a project, %w", err)
	}

	if err := runtime.StoreProjectMeta(projectDir, meta); err != nil {
		return fmt.Errorf("failed to save project metadata locally, %w", err)
	}

	utils.Logger.Println(styles.Greenf("\nProject %s created successfully!", projectName))
	utils.Logger.Println(utils.ProjectNotes(projectName, meta.ID))

	return nil
}

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deta/space/cmd/shared"
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
		Use:   "new [flags]",
		Short: "Create new project",
		Run: func(cmd *cobra.Command, args []string) {
			projectDir, _ := cmd.Flags().GetString("dir")
			blankProject, _ := cmd.Flags().GetBool("blank")
			projectName, _ := cmd.Flags().GetString("name")

			if !cmd.Flags().Changed("name") {
				abs, err := filepath.Abs(projectDir)
				if err != nil {
					shared.Logger.Printf("%sError getting absolute path of project directory: %s", styles.ErrorExclamation, err.Error())
					os.Exit(1)
				}

				name := filepath.Base(abs)
				projectName, err = selectProjectName(name)
				if err != nil {
					os.Exit(1)
				}
			}

			if err := newProject(projectDir, projectName, blankProject); err != nil {
				os.Exit(1)
			}
		},
		PreRunE: shared.CheckAll(
			shared.CheckExists("dir"),
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

	if !shared.IsOutputInteractive() {
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
	res, err := shared.Client.CreateProject(&api.CreateProjectRequest{
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
		return err
	}

	if len(autoDetectedMicros) == 0 {
		_, err := spacefile.CreateBlankSpacefile(projectDir)
		return err
	}

	for _, micro := range autoDetectedMicros {
		shared.Logger.Printf("\nMicro found in \"%s\"", styles.Code(micro.Src))
		shared.Logger.Printf("L engine: %s\n", styles.Blue(micro.Engine))
	}

	if !shared.IsOutputInteractive() {
		_, err = spacefile.CreateSpacefileWithMicros(projectDir, autoDetectedMicros)
		return err
	}

	shared.Logger.Println()
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
			shared.Logger.Printf("failed to create spacefile: %s", err)
			return err
		}
	}

	// add .space folder to gitignore
	if err := runtime.AddSpaceToGitignore(projectDir); err != nil {
		shared.Logger.Printf("failed to add .space to gitignore: %s", err)
		return err
	}

	// Create project
	meta, err := createProject(projectName)
	if err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			shared.Logger.Println(shared.LoginInfo())
			return err
		}
		shared.Logger.Printf("failed to create project: %s", err)
		return err
	}

	if err := runtime.StoreProjectMeta(projectDir, meta); err != nil {
		shared.Logger.Printf("failed to save project meta, %s", err)
		return err
	}

	shared.Logger.Println(styles.Greenf("\nProject %s created successfully!", projectName))
	shared.Logger.Println(shared.ProjectNotes(projectName, meta.ID))

	return nil
}

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/spf13/cobra"
)

func newCmdNew() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [flags]",
		Short: "create new project",
		RunE:  new,
		PreRunE: CheckAll(
			CheckExists("dir"),
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

	if !isOutputInteractive() {
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
	res, err := client.CreateProject(&api.CreateProjectRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	return &runtime.ProjectMeta{ID: res.ID, Name: res.Name, Alias: res.Alias}, nil
}

func new(cmd *cobra.Command, args []string) (err error) {
	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	projectDir, _ := cmd.Flags().GetString("dir")
	blankProject, _ := cmd.Flags().GetBool("blank")
	projectName, _ := cmd.Flags().GetString("name")

	if !cmd.Flags().Changed("name") {
		projectName, err = selectProjectName(filepath.Base(projectDir))
		if err != nil {
			os.Exit(1)
		}
	}

	// Create spacefile if it doesn't exist
	spaceFilePath := path.Join(projectDir, "Spacefile")
	if _, err := os.Stat(spaceFilePath); errors.Is(err, os.ErrNotExist) {
		if blankProject {
			if _, err = spacefile.CreateBlankSpacefile(projectDir); err != nil {
				return fmt.Errorf("failed to create blank project, %w", err)
			}
		} else {
			autoDetectedMicros, err := scanner.Scan(projectDir)
			if err != nil {
				return fmt.Errorf("problem while trying to auto detect runtimes/frameworks for project %s, %w", projectName, err)
			}

			for _, micro := range autoDetectedMicros {
				logger.Printf("Micro found in \"%s\"\n", styles.Code(fmt.Sprintf("%s/", micro.Src)))
				logger.Printf("L engine: %s\n\n", styles.Blue(micro.Engine))
			}

			_, err = spacefile.CreateSpacefileWithMicros(projectDir, autoDetectedMicros)
			if err != nil {
				return fmt.Errorf("failed to create project with detected micros, %w", err)
			}
		}
	}

	// add .space folder to gitignore
	if err := runtime.AddSpaceToGitignore(projectDir); err != nil {
		return fmt.Errorf("failed to add .space to gitignore, %w", err)
	}

	// Create project
	meta, err := createProject(projectName)
	if err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			logger.Println(LoginInfo())
			return nil
		}
		return err
	}

	if err := runtime.StoreProjectMeta(projectDir, meta); err != nil {
		return fmt.Errorf("failed to save project meta, %w", err)
	}

	logger.Println(styles.Greenf("Project %s created successfully!", projectName))

	logger.Println(projectNotes(projectName, meta.ID))

	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}

	return nil
}

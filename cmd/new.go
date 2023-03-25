package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	projectName string
	projectDir  string
	blank       bool

	newCmd = &cobra.Command{
		Use:    "new [flags]",
		Short:  "create new project",
		RunE:   new,
		PreRun: newPreRun,
	}
)

func init() {
	newCmd.Flags().StringVarP(&projectName, "name", "n", "", "project name")
	newCmd.Flags().StringVarP(&projectDir, "dir", "d", "./", "src of project to release")
	newCmd.MarkFlagDirname("dir")
	newCmd.Flags().BoolVarP(&blank, "blank", "b", false, "create blank project")
	rootCmd.AddCommand(newCmd)
}

func projectNameValidator(projectName string) error {

	if len(projectName) < 4 {
		return fmt.Errorf("project name \"%s\" must be at least 4 characters long", projectName)
	}

	if len(projectName) > 16 {
		return fmt.Errorf("project name \"%s\" must be at most 16 characters long", projectName)
	}

	return nil
}

func selectProjectName(placeholder string) (string, error) {

	promptInput := text.Input{
		Prompt:      "What is your project's name?",
		Placeholder: placeholder,
		Validator:   projectNameValidator,
	}

	return text.Run(&promptInput)
}

func createProject(name string, runtimeManager *runtime.Manager) (*runtime.ProjectMeta, error) {
	res, err := client.CreateProject(&api.CreateProjectRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	return &runtime.ProjectMeta{ID: res.ID, Name: res.Name, Alias: res.Alias}, nil
}

func newPreRun(cmd *cobra.Command, args []string) {
	if _, err := os.Stat(path.Join(projectDir, ".space")); !errors.Is(err, os.ErrNotExist) {
		logger.Println(styles.Error("A project already exists in this directory."))
		logger.Println(styles.Error("You can use"), styles.Code("space push"), styles.Error("to create a Revision."))
		os.Exit(1)
	}
}

func new(cmd *cobra.Command, args []string) (err error) {
	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	projectName, _ := cmd.Flags().GetString("name")
	if strings.TrimSpace(projectName) == "" {
		projectName, err = selectProjectName(filepath.Base(projectDir))
		if err != nil {
			return fmt.Errorf("problem while trying to get project's name through prompt, %w", err)
		}
	}

	runtimeManager := runtime.NewManager(projectDir)

	// Create spacefile if it doesn't exist
	spaceFilePath := path.Join(projectDir, "Spacefile")
	if _, err := os.Stat(spaceFilePath); errors.Is(err, os.ErrNotExist) {
		if blank {
			if _, err = spacefile.CreateBlankSpacefile(projectDir); err != nil {
				return fmt.Errorf("failed to create blank project, %w", err)
			}
		} else {
			autoDetectedMicros, err := scanner.Scan(projectDir)
			if err != nil {
				return fmt.Errorf("problem while trying to auto detect runtimes/frameworks for project %s, %w", projectName, err)
			}

			_, err = spacefile.CreateSpacefileWithMicros(projectDir, autoDetectedMicros)
			if err != nil {
				return fmt.Errorf("failed to create project with detected micros, %w", err)
			}

			logDetectedMicros(autoDetectedMicros)
		}
	}

	// add .space folder to gitignore
	if err := runtime.AddSpaceToGitignore(projectDir); err != nil {
		return fmt.Errorf("failed to add .space to gitignore, %w", err)
	}

	// Create project
	meta, err := createProject(projectName, runtimeManager)
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

	logger.Println(styles.Greenf("%s Project", emoji.Check), styles.Pink(projectName), styles.Green("created successfully!"))

	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}

	return nil
}

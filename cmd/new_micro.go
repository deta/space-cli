package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/pkg/choose"
	"github.com/deta/pc-cli/pkg/text"
	"github.com/spf13/cobra"
)

var (
	// flags
	microName       string
	microSrc        string
	microEngine     string
	confirmAddMicro bool

	newMicroCmd = &cobra.Command{
		Use:   "micro",
		Short: "add a new micro",
		RunE:  newMicro,
	}
)

func init() {
	newCmd.AddCommand(newMicroCmd)
	newMicroCmd.Flags().StringVarP(&microName, "name", "n", "", "name of the new micro")
	newMicroCmd.Flags().StringVarP(&microSrc, "src", "s", "", "where is this micro")
	newMicroCmd.Flags().StringVarP(&microEngine, "engine", "e", "", "what type of micro")
	newMicroCmd.Flags().BoolVarP(&confirmAddMicro, "confirm", "c", false, "confirm add new micro, run in non-interactive mode")
}

func selectMicroType() (string, error) {
	promptInput := choose.Input{
		Prompt:  "What type of micro do you want to create?",
		Choices: MicroTypes,
	}

	m, err := choose.Run(&promptInput)
	return MicroTypes[m.Cursor], err
}

func selectFramework(microType string) (string, error) {
	frameworks := MicroTypesToFrameworks[microType]
	promptInput := choose.Input{
		Prompt:  "What framework do you want to use?",
		Choices: frameworks,
	}

	m, err := choose.Run(&promptInput)
	return frameworks[m.Cursor], err
}

func selectMicroName() (string, error) {
	promptInput := text.Input{
		Prompt:      "What do you want to call your micro?",
		Placeholder: "default",
	}

	return text.Run(&promptInput)
}

func selectMicroSrc(microName string) (string, error) {
	promptInput := text.Input{
		Prompt:      "Where do you want to create your micro?",
		Placeholder: microName,
	}

	return text.Run(&promptInput)
}

type addMicroInput struct {
	microName   string
	microSrc    string
	microEngine string
	projectSrc  string
	manifest    *manifest.Manifest
}

func addMicro(i *addMicroInput) error {
	var err error

	microEngine = i.microEngine
	microName = i.microName
	microSrc = i.microSrc

	if !isFlagSet(microEngine) {
		microType, err := selectMicroType()
		if err != nil {
			return fmt.Errorf("error while trying to retrieve micro's type through select prompt, %w", err)
		}

		microEngine, err = selectFramework(microType)
		if err != nil {
			return fmt.Errorf("error while trying to retrieve micro's framework through select prompt, %w", err)
		}
	}

	if !isFlagSet(microName) {
		microName, err = selectMicroName()
		if err != nil {
			return fmt.Errorf("error while trying to retrieve micro's name through text prompt, %w", err)
		}
	}

	if !isFlagSet(microSrc) {
		microSrc, err = selectMicroSrc(microName)
		if err != nil {
			return fmt.Errorf("error while trying to retrieve micro's src through select prompt, %w", err)
		}
	}

	micro := manifest.Micro{
		Name:   microName,
		Src:    microSrc,
		Engine: microEngine,
	}
	err = i.manifest.AddMicro(&micro)
	if err != nil {
		return err
	}

	// TODO: download template files
	// o, err := client.DownloadTemplate(&api.DownloadTemplateRequest{Template: microEngine})
	// if err != nil {
	// 	return err
	// }

	// downloadPath := filepath.Join(i.projectSrc, microSrc)
	// err = fs.UnzipTemplates(o.TemplateFiles, downloadPath, o.TemplatePrefix)
	// if err != nil {
	// 	return fmt.Errorf("failed to unzip template files")
	// }

	err = i.manifest.Save(i.projectSrc)
	if err != nil {
		return fmt.Errorf("failed to add new micro's config to `deta.yml`, %w", err)
	}

	return nil
}

func noProjectLogs(dir string) string {

	msg := `
No project initialized in "%s"
Please create a project using "deta new" before adding a micro`

	return fmt.Sprintf(msg, dir)
}

func newMicro(cmd *cobra.Command, args []string) error {
	path, err := os.Getwd()
	if err != nil {
		logger.Println(fmt.Errorf("cannot read current working dir, %w", err))
		return nil
	}

	m, err := manifest.Open("./")
	if err == manifest.ErrManifestNotFound {
		logger.Println(noProjectLogs(filepath.Base(path)))
		return nil
	}

	err = addMicro(&addMicroInput{
		microName:   microName,
		microSrc:    microSrc,
		microEngine: microEngine,
		projectSrc:  "./",
		manifest:    m,
	})
	if err != nil {
		logger.Printf("Error: failed to add a new micro, %v\n", err)
		return nil
	}

	return nil
}

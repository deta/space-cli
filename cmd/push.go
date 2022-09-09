package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	pushProjectID  string
	pushProjectDir string
	pushTag        string
	pushCmd        = &cobra.Command{
		Use:   "push [flags]",
		Short: "push code for project",
		RunE:  push,
	}
)

func init() {
	pushCmd.Flags().StringVarP(&pushProjectID, "id", "i", "", "project id of project to push")
	pushCmd.Flags().StringVarP(&pushProjectDir, "dir", "d", "./", "src of project to push")
	pushCmd.Flags().StringVarP(&pushTag, "tag", "t", "", "tag to identify revision for this push")
	rootCmd.AddCommand(pushCmd)
}

func selectPushProjectID() (string, error) {
	promptInput := text.Input{
		Prompt:      "What's the project id?",
		Placeholder: "",
		Validator:   projectIDValidator,
	}

	return text.Run(&promptInput)
}

func selectPushTag() (string, error) {
	promptInput := text.Input{
		Prompt:      "Provide a tag for this push",
		Placeholder: "",
	}

	return text.Run(&promptInput)
}

func push(cmd *cobra.Command, args []string) error {

	var err error

	pushProjectDir = filepath.Clean(pushProjectDir)

	runtimeManager, err := runtime.NewManager(&pushProjectDir, false)
	if err != nil {
		return err
	}

	isProjectInitialized, err := runtimeManager.IsProjectInitialized()
	if err != nil {
		return err
	}

	if isProjectInitialized {
		projectMeta, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return err
		}
		pushProjectID = projectMeta.ID
	} else if isFlagEmpty(pushProjectID) {
		logger.Printf("> No project initialized. You can still push by providing a valid project id.\n\n")

		pushProjectID, err = selectPushProjectID()
		if err != nil {
			return fmt.Errorf("problem while trying to get project id to push from text prompt, %w", err)
		}
	}

	isManifestPrsent, err := manifest.IsManifestPresent(pushProjectDir)
	if err != nil {
		return err
	}

	if !isManifestPrsent {
		logger.Println("No manifest present. Please add a manifest before pushing code.")
	}

	if isFlagEmpty(pushTag) {
		pushTag, err = selectPushTag()
		if err != nil {
			return fmt.Errorf("problem while trying to get tag from prompt, %w", err)
		}
	}

	// parse manifest and validate
	logger.Printf("Validating manifest...\n\n")

	m, err := manifest.Open(projectDir)
	if err != nil {
		logger.Printf("Error: %v\n", err)
		return nil
	}
	manifestErrors := scanner.ValidateManifest(m)

	if len(manifestErrors) > 0 {
		logValidationErrors(m, manifestErrors)
		logger.Println(styles.Error.Render("\nPlease try to fix the issues with manifest before pushing code for project."))
		return nil
	} else {
		logger.Printf(styles.Green.Render("Nice! Manifest looks good 🎉!\n\n"))
	}

	logger.Println("Creating a build job....")
	br, err := client.CreateBuild(&api.CreateBuildRequest{AppID: pushProjectID, Tag: "nd"})
	if err != nil {
		return err
	}
	logger.Println("Successfully created build job!")

	logger.Println("Pushing manifest...")
	raw, err := manifest.OpenRaw(pushProjectDir)
	if err != nil {
		return err
	}
	if _, err = client.PushManifest(&api.PushManifestRequest{
		Manifest: raw,
		BuildID:  br.ID,
	}); err != nil {
		return err
	}
	logger.Println("Successfully pushed manifest!")

	logger.Println("Pushing code...")
	zippedCode, err := runtime.ZipDir(pushProjectDir)
	if err != nil {
		return err
	}
	if _, err = client.PushCode(&api.PushCodeRequest{
		BuildID: br.ID, ZippedCode: zippedCode,
	}); err != nil {
		return err
	}

	buildLogs := make(chan string)
	getLogs := func() {
		err = client.GetBuildLogs(&api.GetBuildLogsRequest{BuildID: br.ID}, buildLogs)
		if err != nil {
			logger.Fatal(err)
		}
		close(buildLogs)
	}
	go getLogs()

	for msg := range buildLogs {
		logger.Print(msg)
	}

	logger.Println("Successfully pushed code and created a build artefact!")
	return nil
}

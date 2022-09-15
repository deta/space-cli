package cmd

import (
	"bufio"
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
	pushCmd.Flags().StringVarP(&pushTag, "tag", "t", "", "tag to identify this push")
	rootCmd.AddCommand(pushCmd)
}

func selectPushProjectID() (string, error) {
	promptInput := text.Input{
		Prompt:      "What is your Project ID?",
		Placeholder: "",
		Validator:   projectIDValidator,
	}

	return text.Run(&promptInput)
}

func selectPushTag() (string, error) {
	promptInput := text.Input{
		Prompt:      "Provide a Tag for this push (or leave it empty to auto-generate)",
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
		logger.Printf("> No project was found in the current directory.\n\n")
		logger.Printf("You can still push by providing a valid Project ID.\n\n")

		pushProjectID, err = selectPushProjectID()
		if err != nil {
			return fmt.Errorf("problem while trying to get project id to push from prompt, %w", err)
		}
	}

	isManifestPrsent, err := manifest.IsManifestPresent(pushProjectDir)
	if err != nil {
		return err
	}

	if !isManifestPrsent {
		logger.Println("No Space Manifest is present. Please add a Space Manifest before pushing code.")
	}

	if isFlagEmpty(pushTag) {
		pushTag, err = selectPushTag()
		if err != nil {
			return fmt.Errorf("problem while trying to get tag from prompt, %w", err)
		}
	}

	// parse manifest and validate
	logger.Printf("Validating Space Manifest...\n\n")

	m, err := manifest.Open(projectDir)
	if err != nil {
		logger.Printf("❗ Error: %v\n", err)
		return nil
	}
	manifestErrors := scanner.ValidateManifest(m)

	if len(manifestErrors) > 0 {
		logValidationErrors(m, manifestErrors)
		logger.Println(styles.Error.Render("\nPlease try to fix the issues with your Space Manifest before pushing code."))
		return nil
	} else {
		logger.Printf(styles.Green.Render("Your Space Manifest looks good, proceeding with your push!!\n"))
	}

	logger.Println("⚙️  Working on starting your build ...")
	br, err := client.CreateBuild(&api.CreateBuildRequest{AppID: pushProjectID, Tag: "nd"})
	if err != nil {
		return err
	}
	logger.Println("✅ Successfully started your build!")

	logger.Println("⚙️  Pushing your Space Manifest...")
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
	logger.Println("✅ Successfully pushed your Space Manifest!")

	logger.Println("⚙️  Pushing your code...")
	zippedCode, err := runtime.ZipDir(pushProjectDir)
	if err != nil {
		return err
	}
	if _, err = client.PushCode(&api.PushCodeRequest{
		BuildID: br.ID, ZippedCode: zippedCode,
	}); err != nil {
		return err
	}

	logger.Println("⚙️  Starting your build...")
	readCloser, err := client.GetBuildLogs(&api.GetBuildLogsRequest{
		BuildID: br.ID,
	})
	if err != nil {
		logger.Printf("Error: %v\n", err)
		return nil
	}

	defer readCloser.Close()
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("Error: %v\n", err)
		return nil
	}
	logger.Printf("🎉 Successfully pushed your code and created a new Revision!\n\n")
	logger.Println("Run \"deta release\" to create an installable Release for this Revision.")
	return nil
}

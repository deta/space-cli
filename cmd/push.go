package cmd

import (
	"bufio"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/emoji"
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
	logger.Println()
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
		logger.Printf("%s No project was found in the current directory.\n\n", styles.Info)
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
		logger.Println(styles.Errorf("%s No Space Manifest is present. Please add a Space Manifest before pushing code.", emoji.ErrorExclamation))
		return nil
	}

	// parse manifest and validate
	logger.Printf("Validating Space Manifest...\n\n")

	m, err := manifest.Open(projectDir)
	if err != nil {
		logger.Println(styles.Error(fmt.Sprintf("%s Error: %v", emoji.ErrorExclamation, err)))
		return nil
	}
	manifestErrors := scanner.ValidateManifest(m)

	if len(manifestErrors) > 0 {
		logValidationErrors(m, manifestErrors)
		logger.Println(styles.Error("\nPlease try to fix the issues with your Space Manifest before pushing code."))
		return nil
	} else {
		logValidationErrors(m, manifestErrors)
		logger.Printf(styles.Green("\nYour Space Manifest looks good, proceeding with your push!!\n"))
	}

	logger.Printf("%s Working on starting your build ...\n", emoji.Package)
	br, err := client.CreateBuild(&api.CreateBuildRequest{AppID: pushProjectID})
	if err != nil {
		return err
	}
	logger.Printf("%s Successfully started your build!\n", emoji.Check)

	logger.Printf("%s Pushing your Space Manifest...\n", emoji.Package)
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
	logger.Printf("%s Successfully pushed your Space Manifest!\n", emoji.Check)

	logger.Printf("%s Pushing your code ...\n", emoji.Package)
	zippedCode, err := runtime.ZipDir(pushProjectDir)
	if err != nil {
		return err
	}
	if _, err = client.PushCode(&api.PushCodeRequest{
		BuildID: br.ID, ZippedCode: zippedCode,
	}); err != nil {
		return err
	}

	logger.Printf("%s Starting your build...", emoji.Check)
	readCloser, err := client.GetBuildLogs(&api.GetBuildLogsRequest{
		BuildID: br.ID,
	})
	if err != nil {
		logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return nil
	}

	defer readCloser.Close()
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return nil
	}
	logger.Println(styles.Greenf("\n%s Successfully pushed your code and created a new Revision!\n", emoji.PartyPopper))
	logger.Printf("Run %s to create an installable Release for this Revision.\n", styles.Code("deta release"))
	return nil
}

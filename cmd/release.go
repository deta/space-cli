package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/choose"
	"github.com/deta/pc-cli/pkg/components/confirm"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/spf13/cobra"
)

var (
	releaseDir       string
	revisionID       string
	releaseProjectID string
	releaseVersion   string
	releaseDesc      string

	releaseCmd = &cobra.Command{
		Use:   "release [flags]",
		Short: "create release for a project",
		RunE:  release,
	}
)

func init() {
	releaseCmd.Flags().StringVarP(&releaseDir, "dir", "d", "./", "src of project to release")
	releaseCmd.Flags().StringVarP(&releaseProjectID, "id", "i", "", "project id of an existing project")
	releaseCmd.Flags().StringVarP(&revisionID, "rid", "r", "", "revision id for release")
	releaseCmd.Flags().StringVarP(&releaseVersion, "version", "v", "", "version for the release")
	releaseCmd.Flags().StringVarP(&releaseDesc, "short-desc", "s", "", "ashort description for the release")
	rootCmd.AddCommand(releaseCmd)
}

func selectProjectID() (string, error) {
	promptInput := text.Input{
		Prompt:      "What is your Project ID?",
		Placeholder: "",
		Validator:   emptyPromptValidator,
	}

	return text.Run(&promptInput)
}

func selectRevision(revisions []*api.Revision) (*api.Revision, error) {
	tags := []string{}
	for _, revision := range revisions {
		tags = append(tags, revision.Tag)
	}

	m, err := choose.Run(&choose.Input{
		Prompt:  "Choose a revision.",
		Choices: tags,
	})
	if err != nil {
		return nil, err
	}

	return revisions[m.Cursor], nil
}

func release(cmd *cobra.Command, args []string) error {

	releaseDir = filepath.Clean(releaseDir)

	runtimeManager, err := runtime.NewManager(&releaseDir, true)
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
		releaseProjectID = projectMeta.ID
	} else if isFlagEmpty(releaseProjectID) {
		logger.Printf("> No project was found locally. You can still create a Release by providing a valid Project ID.\n\n")

		releaseProjectID, err = selectProjectID()
		if err != nil {
			return fmt.Errorf("problem while trying to get project id to release from text prompt, %w", err)
		}
	}

	if isFlagEmpty(revisionID) {
		r, err := client.GetRevisions(&api.GetRevisionsRequest{ID: releaseProjectID})
		if err != nil {
			return err
		}
		latestRevision := r.Revisions[0]

		useLatestRevision, err := confirm.Run(&confirm.Input{
			Prompt: fmt.Sprintf("Do you want to use the latest revision (%s)? (y/n)", latestRevision.Tag),
		})
		if err != nil {
			return fmt.Errorf("problem while trying to get confirmation to use latest revision for this release from prompt, %w", err)
		}

		if !useLatestRevision {
			latestRevision, err = selectRevision(r.Revisions)
			if err != nil {
				return fmt.Errorf("problem while trying to get latest revision from prompt, %w", err)
			}
		}

		revisionID = latestRevision.ID
	}

	// TODO: start promotion
	// TODO: promotion logs
	logger.Println("⚙️  Creating a Release...")
	cr, err := client.CreateRelease(&api.CreateReleaseRequest{
		RevisionID:  revisionID,
		AppID:       releaseProjectID,
		Version:     releaseVersion,
		Description: releaseDesc,
	})
	if err != nil {
		return err
	}

	logs := make(chan string)
	go func() {
		err = client.GetReleaseLogs(&api.GetReleaseLogsRequest{ID: cr.ID}, logs)
		if err != nil {
			logger.Fatal(err)
		}
		close(logs)
	}()

	for msg := range logs {
		logger.Print(msg)
	}

	logger.Println("🚀 Lift off -- successfully created a new Release!")
	logger.Println("🌍 Your Release is available globally on 5 Deta Edges")
	logger.Println("🥳 Anyone can install their own copy of your app.")
	return nil
}

package cmd

import (
	"bufio"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/choose"
	"github.com/deta/pc-cli/pkg/components/confirm"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/spf13/cobra"
)

const (
	ReleaseChannelExp = "experimental"
)

var (
	releaseDir       string
	revisionID       string
	releaseProjectID string
	releaseVersion   string

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
	if len(revisions) > 5 {
		revisions = revisions[:5]
	}
	for _, revision := range revisions {
		tags = append(tags, revision.Tag)
	}

	m, err := choose.Run(&choose.Input{
		Prompt:  fmt.Sprintf("Choose a revision %s:", styles.Subtle("(most recent revisions)")),
		Choices: tags,
	})
	if err != nil {
		return nil, err
	}

	return revisions[m.Cursor], nil
}

func release(cmd *cobra.Command, args []string) error {
	logger.Println()
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
		logger.Printf("No project was found locally. You can still create a Release by providing a valid Project ID.\n\n")

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

		if len(r.Revisions) == 0 {
			logger.Printf(styles.Errorf("%s No revisions found. Please create a revision by running %s", emoji.ErrorExclamation, styles.Code("space push")))
			return nil
		}

		latestRevision := r.Revisions[0]

		useLatestRevision, err := confirm.Run(&confirm.Input{
			Prompt: fmt.Sprintf("Do you want to use the latest revision (%s)? (y/n)", latestRevision.Tag),
		})
		if err != nil {
			return fmt.Errorf("problem while trying to get confirmation to use latest revision for this release from prompt, %w", err)
		}

		revisionID = latestRevision.ID

		if !useLatestRevision {
			selectedRevision, err := selectRevision(r.Revisions)
			if err != nil {
				return fmt.Errorf("problem while trying to get latest revision from prompt, %w", err)
			}
			revisionID = selectedRevision.ID
		}
	}

	// TODO: start promotion
	// TODO: promotion logs
	logger.Printf("%s Creating a Release ...\n\n", emoji.Package)
	cr, err := client.CreateRelease(&api.CreateReleaseRequest{
		RevisionID:  revisionID,
		AppID:       releaseProjectID,
		Version:     releaseVersion,
		Channel:     ReleaseChannelExp, // always experimental release for now
	})
	if err != nil {
		return err
	}
	readCloser, err := client.GetReleaseLogs(&api.GetReleaseLogsRequest{
		ID: cr.ID,
	})
	if err != nil {
		logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		return nil
	}

	defer readCloser.Close()
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return nil
	}

	r, err := client.GetReleasePromotion(&api.GetReleasePromotionRequest{PromotionID: cr.ID})
	if err != nil {
		logger.Printf(styles.Errorf("\n%s Failed to check if release succeded. Please check %s if a new release was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, releaseProjectID)))
		return nil
	}

	if r.Status == api.Complete {
		logger.Println()
		logger.Println(emoji.Rocket, "Lift off -- successfully created a new Release!")
		logger.Println(emoji.Earth, "Your Release is available globally on 5 Deta Edges")
		logger.Println(emoji.PartyFace, "Anyone can install their own copy of your app.")
	} else {
		logger.Println(styles.Errorf("\n%s Failed to create release. Please try again!", emoji.ErrorExclamation))
	}

	return nil
}

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/pkg/components/choose"
	"github.com/deta/pc-cli/pkg/components/confirm"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/pkg/components/textarea"
	"github.com/spf13/cobra"
)

const (
	ReleaseChannelExp = "experimental"
)

var (
	releaseDir        string
	revisionID        string
	releaseProjectID  string
	releaseVersion    string
	releaseNotes      string
	listedRelease     bool
	useLatestRevision bool

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
	releaseCmd.Flags().BoolVarP(&listedRelease, "listed", "l", false, "listed on discovery")
	releaseCmd.Flags().BoolVarP(&useLatestRevision, "confirm", "c", false, "release latest revision")
	releaseCmd.Flags().StringVarP(&releaseNotes, "notes", "n", "", "release notes")
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
	if !isOutputInteractive() {
		return nil, fmt.Errorf("cannot select revision in non-interactive mode")
	}
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

func selectReleaseNotes() (string, error) {
	notes, err := textarea.Run(&textarea.Input{
		Placeholder: "start typing...",
		Prompt:      "Enter your Release notes.",
	})
	return notes, err
}

func release(cmd *cobra.Command, args []string) (err error) {
	logger.Println()

	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	releaseDir = filepath.Clean(releaseDir)
	isProjectInitialized := runtime.IsProjectInitialized(releaseDir)

	if isProjectInitialized {
		projectMeta, err := runtime.GetProjectMeta(releaseDir)
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

	if !cmd.Flags().Changed("notes") {
		if !isOutputInteractive() {
			return fmt.Errorf("--notes flag is required in non-interactive mode")
		}
		releaseNotes, err = selectReleaseNotes()
		if err != nil {
			return fmt.Errorf("problem while trying to get release notes from text area, %w", err)
		}
	} else if !isFlagEmpty(releaseNotes) {
		logger.Printf("Using notes provided via arguments.\n\n")
	}

	if isFlagEmpty(revisionID) {
		r, err := client.GetRevisions(&api.GetRevisionsRequest{ID: releaseProjectID})
		if err != nil {
			if errors.Is(err, auth.ErrNoAccessTokenFound) {
				logger.Println(LoginInfo())
				return nil
			} else {
				logger.Println(styles.Errorf("%s Failed to get revisions: %v", emoji.ErrorExclamation, err))
				return nil
			}
		}

		if len(r.Revisions) == 0 {
			logger.Printf(styles.Errorf("%s No revisions found. Please create a revision by running %s", emoji.ErrorExclamation, styles.Code("space push")))
			return nil
		}

		latestRevision := r.Revisions[0]

		if !useLatestRevision {
			if !isOutputInteractive() {
				return fmt.Errorf("--confirm or --rid flag is required in non-interactive mode")
			}
			useLatestRevision, err = confirm.Run(&confirm.Input{
				Prompt: fmt.Sprintf("Do you want to use the latest revision (%s)? (y/n)", latestRevision.Tag),
			})
			if err != nil {
				return fmt.Errorf("problem while trying to get confirmation to use latest revision for this release from prompt, %w", err)
			}
		}

		revisionID = latestRevision.ID

		if !useLatestRevision {
			if !isOutputInteractive() {
				selectedRevision, err := selectRevision(r.Revisions)
				if err != nil {
					return fmt.Errorf("problem while trying to get latest revision from prompt, %w", err)
				}
				revisionID = selectedRevision.ID
			}
		}
	}

	logger.Printf(getCreatingReleaseMsg(listedRelease, useLatestRevision))
	cr, err := client.CreateRelease(&api.CreateReleaseRequest{
		RevisionID:    revisionID,
		AppID:         releaseProjectID,
		Version:       releaseVersion,
		ReleaseNotes:  releaseNotes,
		DiscoveryList: listedRelease,
		Channel:       ReleaseChannelExp, // always experimental release for now
	})
	if err != nil {
		if errors.Is(err, auth.ErrNoAccessTokenFound) {
			logger.Println(LoginInfo())
			return nil
		}
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
		logger.Printf(styles.Errorf("\n%s Failed to check if release succeeded. Please check %s if a new release was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, releaseProjectID)))
		return nil
	}

	if r.Status == api.Complete {
		logger.Println()
		logger.Println(emoji.Rocket, "Lift off -- successfully created a new Release!")
		logger.Println(emoji.Earth, "Your Release is available globally on 5 Deta Edges")
		logger.Println(emoji.PartyFace, "Anyone can install their own copy of your app.")
		if listedRelease {
			logger.Println(emoji.CrystalBall, "Listed on Discovery for others to find!")
		}
		cm := <-c
		if cm.err == nil && cm.isLower {
			logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
		}
	} else {
		logger.Println(styles.Errorf("\n%s Failed to create release. Please try again!", emoji.ErrorExclamation))
	}

	return nil
}

func getCreatingReleaseMsg(listed bool, latest bool) string {
	var listedInfo string
	var latestInfo string
	if listed {
		listedInfo = " listed"
	}
	if latest {
		latestInfo = " with the latest Revision"
	}
	return fmt.Sprintf("%s Creating a%s Release%s ...\n\n", emoji.Package, listedInfo, latestInfo)
}

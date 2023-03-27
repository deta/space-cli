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
	"github.com/deta/pc-cli/pkg/components/textarea"
	"github.com/spf13/cobra"
)

const (
	ReleaseChannelExp = "experimental"
)

func newCmdRelease() *cobra.Command {
	checkNonInteractive := func(cmd *cobra.Command, args []string) error {
		if isOutputInteractive() {
			return nil
		}

		// check if notes are provided
		if cmd.Flags().Changed("notes") {
			return fmt.Errorf("release notes must be provided in non-interactive mode")
		}

		if !cmd.Flags().Changed("rid") && !cmd.Flags().Changed("latest") {
			return fmt.Errorf("revision id or latest flag must be provided in non-interactive mode")
		}

		return nil
	}

	cmd := &cobra.Command{
		Use:     "release [flags]",
		Short:   "create release for a project",
		RunE:    release,
		PreRunE: CheckAll(CheckProjectInitialized("dir"), CheckNotEmpty("id", "rid", "notes", "versions"), checkNonInteractive),
	}

	cmd.Flags().StringP("dir", "d", "./", "src of project to release")
	cmd.Flags().StringP("id", "i", "", "project id of an existing project")
	cmd.Flags().String("rid", "", "revision id for release")
	cmd.Flags().StringP("version", "v", "", "version for the release")
	cmd.Flags().Bool("listed", false, "listed on discovery")
	cmd.Flags().Bool("latest", false, "release latest revision")
	cmd.Flags().StringP("notes", "n", "", "release notes")

	cmd.MarkFlagsMutuallyExclusive("latest", "rid")

	return cmd
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

func selectReleaseNotes() (string, error) {
	notes, err := textarea.Run(&textarea.Input{
		Placeholder: "start typing...",
		Prompt:      "Enter your Release notes.",
	})
	return notes, err
}

func release(cmd *cobra.Command, args []string) (err error) {
	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	projectDir, _ := cmd.Flags().GetString("dir")
	projectID, _ := cmd.Flags().GetString("id")
	releaseNotes, _ := cmd.Flags().GetString("notes")
	revisionID, _ := cmd.Flags().GetString("rid")
	useLatestRevision, _ := cmd.Flags().GetBool("confirm")
	listedRelease, _ := cmd.Flags().GetBool("listed")
	releaseVersion, _ := cmd.Flags().GetString("version")

	projectDir = filepath.Clean(projectDir)

	if !cmd.Flags().Changed("id") {
		projectMeta, err := runtime.GetProjectMeta(projectDir)
		if err != nil {
			return err
		}
		projectID = projectMeta.ID
	}

	if !cmd.Flags().Changed("notes") {
		releaseNotes, err = selectReleaseNotes()
		if err != nil {
			return fmt.Errorf("problem while trying to get release notes from text area: %w", err)
		}
	}

	if !cmd.Flags().Changed("rid") {
		r, err := client.GetRevisions(&api.GetRevisionsRequest{ID: projectID})
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
			useLatestRevision, err = confirm.Run(&confirm.Input{
				Prompt: fmt.Sprintf("Do you want to use the latest revision (%s)? (y/n)", latestRevision.Tag),
			})
			if err != nil {
				return fmt.Errorf("problem while trying to get confirmation to use latest revision for this release from prompt, %w", err)
			}
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

	logger.Printf(getCreatingReleaseMsg(listedRelease, useLatestRevision))
	cr, err := client.CreateRelease(&api.CreateReleaseRequest{
		RevisionID:    revisionID,
		AppID:         projectID,
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
		logger.Printf(styles.Errorf("\n%s Failed to check if release succeeded. Please check %s if a new release was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, projectID)))
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

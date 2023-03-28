package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/deta/pc-cli/cmd/shared"
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
		if shared.IsOutputInteractive() {
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
		Use:   "release [flags]",
		Short: "create release for a project",
		Run: func(cmd *cobra.Command, args []string) {
			var err error

			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")
			releaseNotes, _ := cmd.Flags().GetString("notes")
			revisionID, _ := cmd.Flags().GetString("rid")
			useLatestRevision, _ := cmd.Flags().GetBool("confirm")
			listedRelease, _ := cmd.Flags().GetBool("listed")
			releaseVersion, _ := cmd.Flags().GetString("version")

			if !cmd.Flags().Changed("id") {
				projectMeta, err := runtime.GetProjectMeta(projectDir)
				if err != nil {
					os.Exit(1)
				}
				projectID = projectMeta.ID
			}

			if !cmd.Flags().Changed("rid") {
				if !cmd.Flags().Changed("confirm") {
					useLatestRevision, err = confirm.Run(&confirm.Input{
						Prompt: "Do you want to use the latest revision? (y/n)",
					})
					if err != nil {
						os.Exit(1)
					}
				}

				revision, err := selectRevision(projectID, useLatestRevision)
				if err != nil {
					os.Exit(1)
				}
				shared.Logger.Printf("Selected revision: %s", styles.Blue(revision.Tag))

				revisionID = revision.ID

			}

			shared.Logger.Printf(getCreatingReleaseMsg(listedRelease, useLatestRevision))
			if err := release(projectDir, projectID, revisionID, releaseVersion, listedRelease, releaseNotes); err != nil {
				os.Exit(1)
			}
		},
		PreRunE: shared.CheckAll(shared.CheckProjectInitialized("dir"), shared.CheckNotEmpty("id", "rid", "notes", "versions"), checkNonInteractive),
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

func selectRevision(projectID string, useLatestRevision bool) (revision *api.Revision, err error) {
	r, err := shared.Client.GetRevisions(&api.GetRevisionsRequest{ID: projectID})
	if err != nil {
		if errors.Is(err, auth.ErrNoAccessTokenFound) {
			shared.Logger.Println(shared.LoginInfo())
			return nil, err
		} else {
			shared.Logger.Println(styles.Errorf("%s Failed to get revisions: %v", emoji.ErrorExclamation, err))
			return nil, err
		}
	}
	revisions := r.Revisions

	if len(r.Revisions) == 0 {
		shared.Logger.Printf(styles.Errorf("%s No revisions found. Please create a revision by running %s", emoji.ErrorExclamation, styles.Code("space push")))
		return nil, err
	}

	latestRevision := r.Revisions[0]
	if useLatestRevision {
		return latestRevision, nil
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

func release(projectDir string, projectID string, revisionID string, releaseVersion string, listedRelease bool, releaseNotes string) (err error) {
	cr, err := shared.Client.CreateRelease(&api.CreateReleaseRequest{
		RevisionID:    revisionID,
		AppID:         projectID,
		Version:       releaseVersion,
		ReleaseNotes:  releaseNotes,
		DiscoveryList: listedRelease,
		Channel:       ReleaseChannelExp, // always experimental release for now
	})
	if err != nil {
		if errors.Is(err, auth.ErrNoAccessTokenFound) {
			shared.Logger.Println(shared.LoginInfo())
			return nil
		}
		shared.Logger.Println(styles.Errorf("%s Failed to create release: %v", emoji.ErrorExclamation, err))
		return err
	}
	readCloser, err := shared.Client.GetReleaseLogs(&api.GetReleaseLogsRequest{
		ID: cr.ID,
	})
	if err != nil {
		shared.Logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		return err
	}

	defer readCloser.Close()
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	if err := scanner.Err(); err != nil {
		shared.Logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return err
	}

	r, err := shared.Client.GetReleasePromotion(&api.GetReleasePromotionRequest{PromotionID: cr.ID})
	if err != nil {
		shared.Logger.Printf(styles.Errorf("\n%s Failed to check if release succeeded. Please check %s if a new release was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", shared.BuilderUrl, projectID)))
		return err
	}

	if r.Status == api.Complete {
		shared.Logger.Println()
		shared.Logger.Println(emoji.Rocket, "Lift off -- successfully created a new Release!")
		shared.Logger.Println(emoji.Earth, "Your Release is available globally on 5 Deta Edges")
		shared.Logger.Println(emoji.PartyFace, "Anyone can install their own copy of your app.")
		if listedRelease {
			shared.Logger.Println(emoji.CrystalBall, "Listed on Discovery for others to find!")
		}
	} else {
		shared.Logger.Println(styles.Errorf("\n%s Failed to create release. Please try again!", emoji.ErrorExclamation))
		return fmt.Errorf("release failed: %s", r.Status)
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
	return fmt.Sprintf("\n%s Creating a%s Release%s ...\n\n", emoji.Package, listedInfo, latestInfo)
}

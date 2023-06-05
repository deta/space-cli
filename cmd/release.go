package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/auth"
	"github.com/deta/space/internal/discovery"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/choose"
	"github.com/deta/space/pkg/components/confirm"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/deta/space/pkg/components/text"
	"github.com/deta/space/pkg/util/fs"
	"github.com/deta/space/shared"
	"github.com/spf13/cobra"
)

const (
	ReleaseChannelExp = "experimental"
)

func newCmdRelease() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "release [flags]",
		Short:    "Create a new release from a revision",
		PreRunE:  utils.CheckAll(utils.CheckProjectInitialized("dir"), utils.CheckNotEmpty("id", "rid", "version")),
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			if !utils.IsOutputInteractive() && !cmd.Flags().Changed("rid") && !cmd.Flags().Changed("confirm") {
				utils.Logger.Printf("revision id or confirm flag must be provided in non-interactive mode")
				return err
			}

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
					return err
				}
				projectID = projectMeta.ID
			}

			latestRelease, err := utils.Client.GetLatestReleaseByApp(projectID)
			if err != nil {
				if !errors.Is(err, api.ErrReleaseNotFound) {
					utils.Logger.Println(styles.Errorf("%s Failed to fetch releases: %v", emoji.ErrorExclamation, err))
					return err
				}
			}

			// check there are no previous releases for this project
			if latestRelease == nil && utils.IsOutputInteractive() {
				if !cmd.Flags().Changed("confirm") {
					continueReleasing, err := confirmReleasing(listedRelease)
					if err != nil {
						return err
					}

					if !continueReleasing {
						utils.Logger.Println("Aborted releasing this app.")
						return err
					}
				}
			}

			discoveryData, err := getDiscoveryData(projectDir)
			if err != nil {
				utils.Logger.Printf("Failed to get discovery data: %v", err)
				return err
			}

			if latestRelease != nil {
				err := compareDiscoveryData(discoveryData, latestRelease, projectDir)
				if err != nil {
					return err
				}
			}

			if !cmd.Flags().Changed("rid") {
				if !cmd.Flags().Changed("confirm") {
					useLatestRevision, err = confirm.Run("Do you want to use the latest revision?")
					if err != nil {
						return err
					}
				}

				revision, err := selectRevision(projectID, useLatestRevision)
				if err != nil {
					return err
				}
				utils.Logger.Printf("\nSelected revision: %s", styles.Blue(revision.Tag))

				revisionID = revision.ID

			}

			utils.Logger.Printf(getCreatingReleaseMsg(listedRelease, useLatestRevision))
			if err := release(projectDir, projectID, revisionID, releaseVersion, listedRelease, releaseNotes, discoveryData); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP("dir", "d", "./", "src of project to release")
	cmd.Flags().StringP("id", "i", "", "project id of an existing project")
	cmd.Flags().String("rid", "", "revision id for release")
	cmd.Flags().StringP("version", "v", "", "version for the release")
	cmd.Flags().Bool("listed", false, "listed on discovery")
	cmd.Flags().Bool("confirm", false, "confirm to use latest revision")
	cmd.Flags().StringP("notes", "n", "", "release notes")

	cmd.MarkFlagsMutuallyExclusive("confirm", "rid")

	return cmd
}

func confirmReleasing(listedRelease bool) (bool, error) {
	var confirmMsg string

	if listedRelease {
		utils.Logger.Println("Creating a listed release makes your app available on Deta Discovery for anyone to install and use.")
		confirmMsg = "Are you sure you want to release this app publicly on Discovery? (y/n)"
	} else {
		utils.Logger.Println("Releasing makes your app available via a unique link for others to install and use.")
		confirmMsg = "Are you sure you want to release this app to others? (y/n)"
	}
	utils.Logger.Printf("If you only want to use this app yourself, use your Builder instance instead.\n\n")

	continueReleasing, err := confirm.Run(confirmMsg)
	if err != nil {
		return false, fmt.Errorf("problem while trying to get confirmation to continue releasing this project, %w", err)
	}

	return continueReleasing, nil
}

func promptForDiscoveryData() (*shared.DiscoveryData, error) {
	discoveryData := &shared.DiscoveryData{}

	utils.Logger.Printf("\nPlease give your app a friendly name and add a short description so others know what this app does.\n\n")
	name, err := text.Run(&text.Input{
		Prompt:      "App Name (max 12 chars)",
		Placeholder: "",
		Validator:   validateAppName,
	})
	if err != nil {
		return nil, fmt.Errorf("problem while trying to get title from text prompt, %w", err)
	}
	discoveryData.AppName = name

	tagline, err := text.Run(&text.Input{
		Prompt:      "Short Description (max 69 chars)",
		Placeholder: "",
		Validator:   validateAppDescription,
	})
	if err != nil {
		return nil, fmt.Errorf("problem while trying to get tagline from text prompt, %w", err)
	}
	discoveryData.Tagline = tagline

	return discoveryData, nil
}

func validatePromptValue(value string, min int, max int) error {
	if len(value) < min {
		return fmt.Errorf("must be at least %v characters long", min)
	}

	if len(value) > max {
		return fmt.Errorf("must be at most %v characters long", max)
	}

	return nil
}

func validateAppName(value string) error {
	return validatePromptValue(value, 4, 12)
}

func validateAppDescription(value string) error {
	return validatePromptValue(value, 4, 69)
}

func compareDiscoveryData(discoveryData *shared.DiscoveryData, latestRelease *api.Release, projectDir string) error {
	if latestRelease.Discovery.ContentRaw != "" && !reflect.DeepEqual(latestRelease.Discovery, discoveryData) {
		p := filepath.Join(projectDir, discovery.DiscoveryFilename)
		modTime, err := fs.GetFileLastChanged(p)
		if err != nil {
			utils.Logger.Println(styles.Errorf("%s Failed to check if local Discovery data is outdated: %v", emoji.ErrorExclamation, err))
			return err
		}

		parsedTime, err := time.Parse(time.RFC3339, latestRelease.ReleasedAt)
		if err != nil {
			utils.Logger.Println(styles.Errorf("%s Failed to check if local Discovery data is outdated: %v", emoji.ErrorExclamation, err))
			return err
		}

		if modTime.Before(parsedTime) {
			utils.Logger.Print("\nWarning: your local Discovery data is different from the latest release's Discovery data.\n\n")

			updateLocalDiscovery, err := confirm.Run("Do you want to update your local Discovery.md file with the data from the latest release?")
			if err != nil {
				utils.Logger.Println("Aborted releasing this app.")
				return err
			}

			if updateLocalDiscovery {
				discoveryData = latestRelease.Discovery
				discoveryPath := filepath.Join(projectDir, discovery.DiscoveryFilename)
				err := discovery.CreateDiscoveryFile(discoveryPath, *discoveryData)
				if err != nil {
					utils.Logger.Println(styles.Errorf("%s Failed to update local Discovery.md file: %v", emoji.ErrorExclamation, err))
					return err
				}

				utils.Logger.Printf("\n%s Updated your local Discovery.md file with the latest data!\n\n", emoji.Check)
			} else {
				continueReleasing, err := confirm.Run("Are you sure you want to continue releasing the app with the local Discovery data?")
				if err != nil {
					utils.Logger.Println("Aborted releasing this app.")
					return err
				} else if !continueReleasing {
					utils.Logger.Println("Aborted releasing this app.")
					return fmt.Errorf("aborted releasing this app")
				}
			}
		}
	}

	return nil
}

func getDiscoveryData(projectDir string) (*shared.DiscoveryData, error) {
	discoveryPath := filepath.Join(projectDir, discovery.DiscoveryFilename)
	if _, err := os.Stat(discoveryPath); os.IsNotExist(err) {
		if !utils.IsOutputInteractive() {
			return &shared.DiscoveryData{}, nil
		}
		discoveryData, err := promptForDiscoveryData()
		if err != nil {
			utils.Logger.Printf("%s Error: %v", emoji.ErrorExclamation, err)
		}
		err = discovery.CreateDiscoveryFile(discoveryPath, *discoveryData)
		if err != nil {
			utils.Logger.Printf("%s Failed to create Discovery.md file, %v", emoji.ErrorExclamation, err)
			return nil, err
		}

		utils.Logger.Printf("\n%s Created a new Discovery.md file that stores this data!\n\n", emoji.Check)

		return discoveryData, nil
	} else if err != nil {
		return nil, err
	}

	df, err := os.ReadFile(discoveryPath)
	if err != nil {
		return nil, err
	}

	discoveryData := &shared.DiscoveryData{}
	rest, err := frontmatter.Parse(bytes.NewReader(df), &discoveryData)
	if err != nil {
		utils.Logger.Println(styles.Errorf("\n%s Failed to parse Discovery file, %v", emoji.ErrorExclamation, err))
		return nil, err
	}

	discoveryData.ContentRaw = string(rest)
	if discoveryData.AppName == "" {
		spacefile, err := spacefile.LoadSpacefile(projectDir)
		if err != nil {
			return nil, err
		}

		utils.Logger.Printf("\nNo app name found in Discovery file. Using the app name from your Spacefile: %s", styles.Code(spacefile.AppName))
		utils.Logger.Printf("Using the app name from your Spacefile is deprecated and will be removed in a future version.\n\n")

		discoveryData.AppName = spacefile.AppName
	}

	return discoveryData, nil
}

func selectRevision(projectID string, useLatestRevision bool) (*api.Revision, error) {
	r, err := utils.Client.GetRevisions(&api.GetRevisionsRequest{ID: projectID})
	if err != nil {
		if errors.Is(err, auth.ErrNoAccessTokenFound) {
			utils.Logger.Println(utils.LoginInfo())
			return nil, err
		} else {
			utils.Logger.Println(styles.Errorf("%s Failed to get revisions: %v", emoji.ErrorExclamation, err))
			return nil, err
		}
	}
	revisions := r.Revisions

	if len(r.Revisions) == 0 {
		utils.Logger.Printf(styles.Errorf("%s No revisions found. Please create a revision by running %s", emoji.ErrorExclamation, styles.Code("space push")))
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

	revisionMap := make(map[string]*api.Revision)
	for _, revision := range revisions {
		revisionMap[revision.Tag] = revision
		tags = append(tags, revision.Tag)
	}

	tag, err := choose.Run(
		fmt.Sprintf("Choose a revision %s:", styles.Subtle("(most recent revisions)")),
		tags...,
	)
	if err != nil {
		return nil, err
	}

	return revisionMap[tag], nil
}

func release(projectDir string, projectID string, revisionID string, releaseVersion string, listedRelease bool, releaseNotes string, discoveryData *shared.DiscoveryData) (err error) {
	cr, err := utils.Client.CreateRelease(&api.CreateReleaseRequest{
		RevisionID:    revisionID,
		AppID:         projectID,
		Version:       releaseVersion,
		ReleaseNotes:  releaseNotes,
		DiscoveryList: listedRelease,
		Channel:       ReleaseChannelExp, // always experimental release for now
	})
	if err != nil {
		if errors.Is(err, auth.ErrNoAccessTokenFound) {
			utils.Logger.Println(utils.LoginInfo())
			return nil
		}
		utils.Logger.Println(styles.Errorf("%s Failed to create release: %v", emoji.ErrorExclamation, err))
		return err
	}

	err = utils.Client.StoreDiscoveryData(cr.ID, discoveryData)
	if err != nil {
		utils.Logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		return err
	}

	readCloser, err := utils.Client.GetReleaseLogs(&api.GetReleaseLogsRequest{
		ID: cr.ID,
	})
	if err != nil {
		utils.Logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		return err
	}

	var releaseUrl string

	defer readCloser.Close()
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "http") {
			releaseUrl = line
		} else {
			fmt.Println(line)
		}
	}
	if err := scanner.Err(); err != nil {
		utils.Logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return err
	}

	r, err := utils.Client.GetReleasePromotion(&api.GetReleasePromotionRequest{PromotionID: cr.ID})
	if err != nil {
		utils.Logger.Printf(styles.Errorf("\n%s Failed to check if release succeeded. Please check %s if a new release was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", utils.BuilderUrl, projectID)))
		return err
	}

	if r.Status == api.Complete {
		utils.Logger.Println()
		utils.Logger.Println(emoji.Rocket, "Lift off -- successfully created a new release!")
		utils.Logger.Println(emoji.Earth, "Your release is available globally on 5 Deta Edges")
		utils.Logger.Println(emoji.PartyFace, "Anyone can install their own copy of your app.")
		if listedRelease {
			utils.Logger.Println(emoji.CrystalBall, "Listed on Discovery for others to find!")
		}

		if releaseUrl != "" {
			utils.Logger.Printf("\nRelease: %s", styles.Code(releaseUrl))
		}
	} else {
		utils.Logger.Println(styles.Errorf("\n%s Failed to create release. Please try again!", emoji.ErrorExclamation))
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
		latestInfo = " with the latest revision"
	}
	return fmt.Sprintf("\n%s Creating a%s release%s ...\n\n", emoji.Package, listedInfo, latestInfo)
}

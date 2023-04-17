package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/auth"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newCmdPush() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [flags]",
		Short: "Push your changes to Space and create a new revision.",
		Long: `Push your changes to Space and create a new revision.

Space will automatically update your Builder instance with the new revision.

If you don't want to follow the logs of the build and update, pass the --skip-logs argument which will exit the process as soon as the build is started instead of waiting for it to finish.

Tip: Use the .spaceignore file to exclude certain files and directories from being uploaded during push.
`,
		Args:     cobra.NoArgs,
		PreRunE:  shared.CheckAll(shared.CheckProjectInitialized("dir"), shared.CheckNotEmpty("id", "tag")),
		PostRunE: shared.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")
			if !cmd.Flags().Changed("id") {
				var err error
				projectID, err = runtime.GetProjectID(projectDir)
				if err != nil {
					shared.Logger.Printf("%s Failed to get project id: %s", emoji.ErrorExclamation, err)
					return err
				}
			}

			pushTag, _ := cmd.Flags().GetString("tag")
			openInBrowser, _ := cmd.Flags().GetBool("open")
			skipLogs, _ := cmd.Flags().GetBool("skip-logs")

			err := push(projectID, projectDir, pushTag, openInBrowser, skipLogs)
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringP("id", "i", "", "project id of project to push")
	cmd.Flags().StringP("dir", "d", "./", "src of project to push")
	cmd.MarkFlagDirname("dir")
	cmd.Flags().StringP("tag", "t", "", "tag to identify this push")
	cmd.Flags().Bool("open", false, "open builder instance/project in browser after push")
	cmd.Flags().BoolP("skip-logs", "", false, "skip following logs after push")

	return cmd
}

func push(projectID string, projectDir string, pushTag string, openInBrowser bool, skipLogs bool) error {
	shared.Logger.Printf("Validating your Spacefile...")

	s, err := spacefile.ParseSpacefile(filepath.Join(projectDir, "Spacefile"))
	if err != nil {
		shared.Logger.Printf("%s Failed to parse Spacefile: %s", emoji.ErrorExclamation, err)
		return err
	}

	shared.Logger.Printf(styles.Green("\nYour Spacefile looks good, proceeding with your push!"))

	// push code & run build steps
	zippedCode, nbFiles, err := runtime.ZipDir(projectDir)
	if err != nil {
		shared.Logger.Printf("%s Failed to zip project: %s", emoji.ErrorExclamation, err)
		return err
	}

	build, err := shared.Client.CreateBuild(&api.CreateBuildRequest{AppID: projectID, Tag: pushTag})
	if err != nil {
		shared.Logger.Printf("%s Failed to push project: %s", emoji.ErrorExclamation, err)
		return err
	}
	shared.Logger.Printf("\n%s Successfully started your build!", emoji.Check)

	// push spacefile
	raw, err := os.ReadFile(filepath.Join(projectDir, "Spacefile"))
	if err != nil {
		shared.Logger.Printf("%s Failed to read Spacefile: %s", emoji.ErrorExclamation, err)
		return err
	}

	_, err = shared.Client.PushSpacefile(&api.PushSpacefileRequest{
		Manifest: raw,
		BuildID:  build.ID,
	})
	if err != nil {
		shared.Logger.Println(styles.Errorf("\n%s Failed to push Spacefile, %v", emoji.ErrorExclamation, err))
		return fmt.Errorf("failed to push Spacefile: %w", err)
	}
	shared.Logger.Printf("%s Successfully pushed your Spacefile!", emoji.Check)

	// // push spacefile icon
	if icon, err := s.GetIcon(); err == nil {
		if _, err := shared.Client.PushIcon(&api.PushIconRequest{
			Icon:        icon.Raw,
			ContentType: icon.IconMeta.ContentType,
			BuildID:     build.ID,
		}); err != nil {
			shared.Logger.Println(styles.Errorf("\n%s Failed to push icon, %v", emoji.ErrorExclamation, err))
			return err
		}
	}

	if _, err = shared.Client.PushCode(&api.PushCodeRequest{
		BuildID: build.ID, ZippedCode: zippedCode,
	}); err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			shared.Logger.Println(shared.LoginInfo())
			return err
		}
		shared.Logger.Printf("%s Failed to push code: %s", emoji.ErrorExclamation, err)
		return err
	}

	shared.Logger.Printf("\n%s Pushing your code (%d files) & running build process...\n", emoji.Package, nbFiles)

	if skipLogs {
		b, err := shared.Client.GetBuild(&api.GetBuildRequest{BuildID: build.ID})
		if err != nil {
			shared.Logger.Printf(styles.Errorf("\n%s Failed to check if build was started. Please check %s for the build status.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", shared.BuilderUrl, projectID)))
			return fmt.Errorf("failed to check if build was started: %w", err)
		}

		var url = fmt.Sprintf("%s/%s?event=bld-%s", shared.BuilderUrl, projectID, b.Tag)

		shared.Logger.Println(styles.Greenf("\n%s Successfully pushed your code!", emoji.PartyPopper))
		shared.Logger.Println("\nSkipped following build process, please check build status manually:")
		shared.Logger.Println(styles.Codef(url))
		if openInBrowser {
			err = browser.OpenURL(url)

			if err != nil {
				shared.Logger.Printf("%s Failed to open browser window", emoji.ErrorExclamation)
				return err
			}
		}

		return nil
	}

	// get build logs
	readCloser, err := shared.Client.GetBuildLogs(&api.GetBuildLogsRequest{
		BuildID: build.ID,
	})
	if err != nil {
		shared.Logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return err
	}
	defer readCloser.Close()
	// stream build logs
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	if err := scanner.Err(); err != nil {
		shared.Logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return err
	}

	// check build status
	b, err := shared.Client.GetBuild(&api.GetBuildRequest{BuildID: build.ID})
	if err != nil {
		shared.Logger.Printf(styles.Errorf("\n%s Failed to check if push succeded. Please check %s if a new revision was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", shared.BuilderUrl, projectID)))
		return err
	}
	if b.Status != api.Complete {
		shared.Logger.Println(styles.Errorf("\n%s Failed to push code and create a revision. Please try again!", emoji.ErrorExclamation))
		return err
	}

	// get promotion via build id (build id == revision id)
	p, err := shared.Client.GetPromotionByRevision(&api.GetPromotionRequest{RevisionID: build.ID})
	if err != nil {
		shared.Logger.Printf(styles.Errorf("\n%s Failed to get promotion. Please check %s if a new revision was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", shared.BuilderUrl, projectID)))
		return err
	}

	shared.Logger.Printf("\n%s Updating your Builder instance with the new revision...\n\n", emoji.Tools)

	readCloserPromotion, err := shared.Client.GetReleaseLogs(&api.GetReleaseLogsRequest{
		ID: p.ID,
	})
	if err != nil {
		shared.Logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		return err
	}

	defer readCloserPromotion.Close()
	scannerPromotion := bufio.NewScanner(readCloserPromotion)
	for scannerPromotion.Scan() {
		// we don't want to print the logs to the terminal
	}
	if err := scannerPromotion.Err(); err != nil {
		shared.Logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return err
	}

	// check promotion status
	p, err = shared.Client.GetReleasePromotion(&api.GetReleasePromotionRequest{PromotionID: p.ID})
	if err != nil {
		shared.Logger.Printf(styles.Errorf("\n%s Failed to check if Builder instance was updated. Please check %s", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", shared.BuilderUrl, projectID)))
		return err
	}
	if p.Status != api.Complete {
		shared.Logger.Println(styles.Errorf("\n%s Failed to update Builder instance. Please try again!", emoji.ErrorExclamation))
		return err
	}

	// get installation via promotion id (promotion id == release id)
	i, err := shared.Client.GetInstallationByRelease(&api.GetInstallationByReleaseRequest{ReleaseID: p.ID})
	if err != nil {
		shared.Logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		shared.Logger.Printf(styles.Errorf("\n%s Failed to get installation. Please check %s if your Builder instance is being updated.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", shared.BuilderUrl, projectID)))
		return err
	}

	readCloserInstallation, err := shared.Client.GetInstallationLogs(&api.GetInstallationLogsRequest{
		ID: i.ID,
	})
	if err != nil {
		shared.Logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		return err
	}

	var instanceUrl string

	defer readCloserInstallation.Close()
	scannerInstallation := bufio.NewScanner(readCloserInstallation)
	for scannerInstallation.Scan() {
		line := scannerInstallation.Text()
		if strings.Contains(line, "http") {
			instanceUrl = line
		} else {
			fmt.Println(line)
		}
	}
	if err := scannerInstallation.Err(); err != nil {
		shared.Logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return err
	}

	// check installation status
	i, err = shared.Client.GetInstallation(&api.GetInstallationRequest{ID: i.ID})
	if err != nil {
		shared.Logger.Printf(styles.Errorf("\n%s Failed to check if Builder instance was updated. Please check %s", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", shared.BuilderUrl, projectID)))
		return err
	}
	if i.Status != api.Complete {
		shared.Logger.Println(styles.Errorf("\n%s Failed to update Builder instance. Please try again!", emoji.ErrorExclamation))
		return err
	}

	shared.Logger.Println(styles.Greenf("\n%s Successfully pushed your code and updated your Builder instance!", emoji.PartyPopper))

	if instanceUrl != "" {
		shared.Logger.Printf("Builder instance: %s", styles.Code(instanceUrl))

		if openInBrowser {
			err = browser.OpenURL(instanceUrl)

			if err != nil {
				shared.Logger.Printf("%s Failed to open browser window", emoji.ErrorExclamation)
				return err
			}
		}
	}

	return nil

}

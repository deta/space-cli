package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/auth"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

const fetchPromotionsRetryCount = 5

func newCmdPush() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [flags]",
		Short: "Push your changes to Space and create a new revision.",
		Long: `Push your changes to Space and create a new revision.

Space will automatically update your Builder instance with the new revision.

If you don't want to follow the logs of the build and update, pass the
--skip-logs argument which will exit the process as soon as the build is started
instead of waiting for it to finish.

Tip: Use the .spaceignore file to exclude certain files and directories from
being uploaded during push.
`,
		Args:     cobra.NoArgs,
		PreRunE:  utils.CheckAll(utils.CheckProjectInitialized("dir"), utils.CheckNotEmpty("id", "tag")),
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")
			if !cmd.Flags().Changed("id") {
				var err error
				projectID, err = runtime.GetProjectID(projectDir)
				if err != nil {
					return fmt.Errorf("failed to get the project id, %w", err)
				}
			}

			pushTag, _ := cmd.Flags().GetString("tag")
			openInBrowser, _ := cmd.Flags().GetBool("open")
			skipLogs, _ := cmd.Flags().GetBool("skip-logs")
			runnerVersion, _ := cmd.Flags().GetString("runner-version")

			return push(projectID, projectDir, pushTag, runnerVersion, openInBrowser, skipLogs)
		},
	}

	cmd.Flags().StringP("id", "i", "", "project id of project to push")
	cmd.Flags().StringP("dir", "d", "./", "src of project to push")
	cmd.MarkFlagDirname("dir")
	cmd.Flags().StringP("tag", "t", "", "tag to identify this push")
	cmd.Flags().Bool("open", false, "open builder instance/project in browser after push")
	cmd.Flags().BoolP("skip-logs", "", false, "skip following logs after push")
	cmd.Flags().StringP("runner-version", "", "", "runner version to use for this push")
	cmd.Flags().MarkHidden("runner-version")

	return cmd
}

func push(projectID, projectDir, pushTag, runnerVersion string, openInBrowser, skipLogs bool) error {
	utils.Logger.Printf("Validating your Spacefile...")

	s, err := spacefile.LoadSpacefile(projectDir)
	if err != nil {
		return fmt.Errorf("failed to parse your Spacefile, %w", err)
	}

	utils.Logger.Printf(styles.Green("\nYour Spacefile looks good, proceeding with your push!"))

	// push code & run build steps
	zippedCode, nbFiles, err := runtime.ZipDir(projectDir)
	if err != nil {
		return fmt.Errorf("failed to zip your project, %w", err)
	}

	build, err := utils.Client.CreateBuild(&api.CreateBuildRequest{AppID: projectID, Tag: pushTag, RunnerVersion: runnerVersion, AutoPWA: *s.AutoPWA})
	if err != nil {
		return fmt.Errorf("failed to start a build, %w", err)
	}
	utils.Logger.Printf("\n%s Successfully started your build!", emoji.Check)

	// push spacefile
	raw, err := os.ReadFile(filepath.Join(projectDir, "Spacefile"))
	if err != nil {
		return fmt.Errorf("failed to read Spacefile, %w", err)
	}

	_, err = utils.Client.PushSpacefile(&api.PushSpacefileRequest{
		Manifest: raw,
		BuildID:  build.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to push Spacefile, %w", err)
	}
	utils.Logger.Printf("%s Successfully pushed your Spacefile!", emoji.Check)

	// // push spacefile icon
	if icon, err := s.GetIcon(); err == nil {
		if _, err := utils.Client.PushIcon(&api.PushIconRequest{
			Icon:        icon.Raw,
			ContentType: icon.IconMeta.ContentType,
			BuildID:     build.ID,
		}); err != nil {
			return fmt.Errorf("failed to push the icon, %w", err)
		}
	}

	if _, err = utils.Client.PushCode(&api.PushCodeRequest{
		BuildID: build.ID, ZippedCode: zippedCode,
	}); err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			utils.Logger.Println(utils.LoginInfo())
			return err
		}
		return fmt.Errorf("failed to push your code, %w", err)
	}

	utils.Logger.Printf("\n%s Pushing your code (%d files) & running build process...\n\n", emoji.Package, nbFiles)

	if skipLogs {
		b, err := utils.Client.GetBuild(&api.GetBuildRequest{BuildID: build.ID})
		if err != nil {
			return fmt.Errorf("failed to check if the build was started, please check %s for the build status", styles.Codef("%s/%s/develop", utils.BuilderUrl, projectID))
		}

		var url = fmt.Sprintf("%s/%s?event=bld-%s", utils.BuilderUrl, projectID, b.Tag)

		utils.Logger.Println(styles.Greenf("\n%s Successfully pushed your code!", emoji.PartyPopper))
		utils.Logger.Println("\nSkipped following build process, please check build status manually:")
		utils.Logger.Println(styles.Codef(url))
		if openInBrowser {
			err = browser.OpenURL(url)

			if err != nil {
				return fmt.Errorf("failed to open a browser window, %w", err)
			}
		}

		return nil
	}

	// get build logs
	readCloser, err := utils.Client.GetBuildLogs(&api.GetBuildLogsRequest{
		BuildID: build.ID,
	})
	if err != nil {
		return err
	}
	defer readCloser.Close()
	// stream build logs
	scanner := bufio.NewScanner(readCloser)
	buildLogger := log.New(os.Stderr, "", 0)
	buildLogger.SetFlags(log.Ldate | log.Ltime)
	for scanner.Scan() {
		line := scanner.Text()
		buildLogger.Println(line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// check build status
	b, err := utils.Client.GetBuild(&api.GetBuildRequest{BuildID: build.ID})
	if err != nil {
		return fmt.Errorf("failed to check if push succeded, please check %s if a new revision was created successfully", styles.Codef("%s/%s/develop", utils.BuilderUrl, projectID))
	}
	if b.Status != api.Complete {
		return fmt.Errorf("failed to push code and create a revision, please try again")
	}

	// get promotion via build id (build id == revision id)
	// loop until either p is not nil, err is not nil, or i is equal to `fetchPromotionRetryCount`
	var p *api.GetReleasePromotionResponse
	for i := 0; i < fetchPromotionsRetryCount; i++ {
		p, err = utils.Client.GetPromotionByRevision(&api.GetPromotionRequest{RevisionID: build.ID})

		if p != nil {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to check if a new revision was created, please check %s manually", styles.Codef("%s/%s/develop", utils.BuilderUrl, projectID))
		}
	}

	utils.Logger.Printf("\n%s Updating your Builder instance with the new revision...\n\n", emoji.Tools)

	readCloserPromotion, err := utils.Client.GetReleaseLogs(&api.GetReleaseLogsRequest{
		ID: p.ID,
	})
	if err != nil {
		return err
	}

	defer readCloserPromotion.Close()
	scannerPromotion := bufio.NewScanner(readCloserPromotion)
	for scannerPromotion.Scan() {
		// we don't want to print the logs to the terminal
	}
	if err := scannerPromotion.Err(); err != nil {
		return err
	}

	// check promotion status
	p, err = utils.Client.GetReleasePromotion(&api.GetReleasePromotionRequest{PromotionID: p.ID})
	if err != nil {
		return fmt.Errorf("failed to check if your Builder instance was updated, please check %s manually", styles.Codef("%s/%s/develop", utils.BuilderUrl, projectID))
	}
	if p.Status != api.Complete {
		return fmt.Errorf("failed to update your Builder instance, please try again")
	}

	// get installation via promotion id (promotion id == release id)
	i, err := utils.Client.GetInstallationByRelease(&api.GetInstallationByReleaseRequest{ReleaseID: p.ID})
	if err != nil {
		return fmt.Errorf("failed to check if your Builder instance is being updated, please check %s manually", styles.Codef("%s/%s/develop", utils.BuilderUrl, projectID))
	}

	readCloserInstallation, err := utils.Client.GetInstallationLogs(&api.GetInstallationLogsRequest{
		ID: i.ID,
	})
	if err != nil {
		return err
	}

	var instanceUrl string

	defer readCloserInstallation.Close()
	scannerInstallation := bufio.NewScanner(readCloserInstallation)

	installationLogger := log.New(os.Stderr, "", 0)
	installationLogger.SetFlags(log.Ldate | log.Ltime)
	for scannerInstallation.Scan() {
		line := scannerInstallation.Text()
		if strings.Contains(line, "http") {
			instanceUrl = line
		} else {
			installationLogger.Println(line)
		}
	}
	if err := scannerInstallation.Err(); err != nil {
		return err
	}

	// check installation status
	i, err = utils.Client.GetInstallation(&api.GetInstallationRequest{ID: i.ID})
	if err != nil {
		return fmt.Errorf("failed to check if your Builder instance was updated, please check %s manually", styles.Codef("%s/%s/develop", utils.BuilderUrl, projectID))
	}
	if i.Status != api.Complete {
		return fmt.Errorf("failed to update your Builder instance, please try again")
	}

	utils.Logger.Println(styles.Greenf("\n%s Successfully pushed your code and updated your Builder instance!", emoji.PartyPopper))

	if instanceUrl != "" {
		utils.Logger.Printf("Builder instance: %s", styles.Code(instanceUrl))

		if openInBrowser {
			err = browser.OpenURL(instanceUrl)

			if err != nil {
				return fmt.Errorf("failed to open a browser window, %w", err)
			}
		}
	}

	return nil

}

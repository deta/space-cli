package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/discovery"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/spinner"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	pushProjectID  string
	pushProjectDir string
	pushTag        string
	pushOpen       bool
	skipLogs       bool
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
	pushCmd.Flags().BoolVarP(&pushOpen, "open", "o", false, "open builder instance/project in browser after push")
	pushCmd.Flags().BoolVarP(&skipLogs, "skip-logs", "", false, "skip following logs after push")
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

func push(cmd *cobra.Command, args []string) error {
	logger.Println()

	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

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

	// check if project is initialized
	if isProjectInitialized {
		projectMeta, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return err
		}
		pushProjectID = projectMeta.ID
	} else if isFlagEmpty(pushProjectID) {
		logger.Printf("No project was found in the current directory.\n\n")
		logger.Printf("You can still push by providing a valid Project ID.\n\n")

		pushProjectID, err = selectPushProjectID()
		if err != nil {
			return fmt.Errorf("problem while trying to get project id to push from prompt, %v", err)
		}
	}

	// check if spacefile is present
	isSpacefilePrsent, err := spacefile.IsSpacefilePresent(pushProjectDir)
	if err != nil {
		if errors.Is(err, spacefile.ErrSpacefileWrongCase) {
			logger.Printf("%s The Spacefile must be called exactly 'Spacefile'.\n", emoji.ErrorExclamation)
			return nil
		}
		return err
	}
	if !isSpacefilePrsent {
		logger.Println(styles.Errorf("%s No Spacefile is present. Please add a Spacefile before pushing code.", emoji.ErrorExclamation))
		return nil
	}

	// parse spacefile and validate
	logger.Printf("Validating Spacefile...\n\n")

	s, err := spacefile.Open(pushProjectDir)
	if err != nil {
		if te, ok := err.(*yaml.TypeError); ok {
			logger.Println(spacefile.ParseSpacefileUnmarshallTypeError(te))
			return nil
		}
		logger.Println(styles.Error(fmt.Sprintf("%s Error: %v", emoji.ErrorExclamation, err)))
		return nil
	}
	spacefileErrors := spacefile.ValidateSpacefile(s, pushProjectDir)

	if len(spacefileErrors) > 0 {
		logValidationErrors(s, spacefileErrors)
		logger.Println(styles.Error("\nPlease try to fix the issues with your Spacefile before pushing code."))
		return nil
	} else {
		logValidationErrors(s, spacefileErrors)
		logger.Printf(styles.Green("\nYour Spacefile looks good, proceeding with your push!!\n"))
	}

	// start push & build process
	buildSpinnerInput := spinner.Input{
		LoadingMsg: "Working on starting your build...",
		Request: func() tea.Msg {
			r, err := client.CreateBuild(&api.CreateBuildRequest{AppID: pushProjectID, Tag: pushTag})

			return spinner.Stop{
				RequestResponse: spinner.RequestResponse{Response: r, Err: err},
				FinishMsg:       fmt.Sprintf("%s Successfully started your build!", emoji.Check),
			}
		},
	}
	r := spinner.Run(&buildSpinnerInput)
	if r.Err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, r.Err) {
			logger.Println(LoginInfo())
			return nil
		}
		logger.Println(styles.Errorf("\n%s Failed to push project: %s", emoji.ErrorExclamation, r.Err))
		return nil
	}
	var br *api.CreateBuildResponse
	var ok bool
	if br, ok = r.Response.(*api.CreateBuildResponse); !ok {
		return fmt.Errorf("failed to parse create build response")
	}

	// push spacefile
	raw, err := spacefile.OpenRaw(pushProjectDir)
	if err != nil {
		return err
	}
	pushSpacefileInput := spinner.Input{
		LoadingMsg: "Pushing your spacefile...",
		Request: func() tea.Msg {
			pr, err := client.PushSpacefile(&api.PushSpacefileRequest{
				Manifest: raw,
				BuildID:  br.ID,
			})
			return spinner.Stop{
				RequestResponse: spinner.RequestResponse{Response: pr, Err: err},
				FinishMsg:       fmt.Sprintf("%s Successfully pushed your Spacefile!", emoji.Check),
			}
		},
	}
	r = spinner.Run(&pushSpacefileInput)
	if r.Err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, r.Err) {
			logger.Println(LoginInfo())
			return nil
		}
		logger.Println(styles.Errorf("\n%s Failed to push Spacefile, %v", emoji.ErrorExclamation, r.Err))
		return nil
	}

	// push spacefile icon
	icon, err := s.GetIcon()
	if err != nil {
		if !errors.Is(err, spacefile.ErrInvalidIconPath) {
			logger.Println(styles.Errorf("\n%s Failed to get icon, %v", emoji.ErrorExclamation, err))
			return nil
		}
	}
	pushSpacefileIcon := spinner.Input{
		LoadingMsg: "Pushing your icon...",
		Request: func() tea.Msg {
			pr, err := client.PushIcon(&api.PushIconRequest{
				Icon:        icon.Raw,
				ContentType: icon.IconMeta.ContentType,
				BuildID:     br.ID,
			})
			return spinner.Stop{
				RequestResponse: spinner.RequestResponse{Response: pr, Err: err},
				FinishMsg:       fmt.Sprintf("%s Successfully pushed your icon!", emoji.Check),
			}
		},
	}
	if !errors.Is(err, spacefile.ErrInvalidIconPath) {
		r = spinner.Run(&pushSpacefileIcon)
		if r.Err != nil {
			if errors.Is(auth.ErrNoAccessTokenFound, r.Err) {
				logger.Println(LoginInfo())
				return nil
			}
			logger.Println(styles.Errorf("\n%s Failed to push icon, %v", emoji.ErrorExclamation, r.Err))
			return nil
		}
	}

	// push discovery file
	df, err := discovery.Open(pushProjectDir)
	if err != nil {
		if !(errors.Is(err, discovery.ErrDiscoveryFileNotFound)) {
			if errors.Is(err, discovery.ErrDiscoveryFileWrongCase) {
				logger.Println(styles.Errorf("\n%s The Discovery file must be called exactly 'Discovery.md'", emoji.ErrorExclamation))
				return nil
			}
			logger.Println(styles.Errorf("\n%s Failed to read Discovery file, %v", emoji.ErrorExclamation, err))
			return nil
		}
	}
	pushDiscoveryFile := spinner.Input{
		LoadingMsg: "Pushing your Discovery file...",
		Request: func() tea.Msg {
			pr, err := client.PushDiscoveryFile(&api.PushDiscoveryFileRequest{
				DiscoveryFile: df,
				BuildID:       br.ID,
			})
			return spinner.Stop{
				RequestResponse: spinner.RequestResponse{Response: pr, Err: err},
				FinishMsg:       fmt.Sprintf("%s Successfully pushed your Discovery file!", emoji.Check),
			}
		},
	}
	if !errors.Is(err, discovery.ErrDiscoveryFileNotFound) {
		r = spinner.Run(&pushDiscoveryFile)
		if r.Err != nil {
			if errors.Is(auth.ErrNoAccessTokenFound, r.Err) {
				logger.Println(LoginInfo())
				return nil
			}
			logger.Println(styles.Errorf("\n%s Failed to push Discovery file, %v", emoji.ErrorExclamation, r.Err))
			return nil
		}
	}

	// push code & run build steps
	logger.Printf("\n%s Pushing your code & running build process...\n", emoji.Package)
	zippedCode, err := runtimeManager.ZipDir(pushProjectDir)
	if err != nil {
		return err
	}
	if _, err = client.PushCode(&api.PushCodeRequest{
		BuildID: br.ID, ZippedCode: zippedCode,
	}); err != nil {
		if errors.Is(auth.ErrNoAccessTokenFound, err) {
			logger.Println(LoginInfo())
			return nil
		}
		return err
	}

	if skipLogs {
		b, err := client.GetBuild(&api.GetBuildRequest{BuildID: br.ID})
		if err != nil {
			logger.Printf(styles.Errorf("\n%s Failed to check if build was started. Please check %s for the build status.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, pushProjectID)))
			return nil
		}

		var url = fmt.Sprintf("%s/%s?event=bld-%s", builderUrl, pushProjectID, b.Tag)

		logger.Println(styles.Greenf("\n%s Successfully pushed your code!", emoji.PartyPopper))
		logger.Println("\nSkipped following build process, please check build status manually:")
		logger.Println(styles.Codef(url))

		if pushOpen {
			err = browser.OpenURL(url)

			if err != nil {
				return fmt.Errorf("%s Failed to open browser window %w", emoji.ErrorExclamation, err)
			}
		}

		cm := <-c
		if cm.err == nil && cm.isLower {
			logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
		}
		return nil
	}

	// get build logs
	readCloser, err := client.GetBuildLogs(&api.GetBuildLogsRequest{
		BuildID: br.ID,
	})
	if err != nil {
		logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return nil
	}
	defer readCloser.Close()
	// stream build logs
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return nil
	}

	// check build status
	b, err := client.GetBuild(&api.GetBuildRequest{BuildID: br.ID})
	if err != nil {
		logger.Printf(styles.Errorf("\n%s Failed to check if push succeded. Please check %s if a new revision was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, pushProjectID)))
		return nil
	}
	if b.Status != api.Complete {
		logger.Println(styles.Errorf("\n%s Failed to push code and create a revision. Please try again!", emoji.ErrorExclamation))
		return nil
	}

	// get promotion via build id (build id == revision id)
	p, err := client.GetPromotionByRevision(&api.GetPromotionRequest{RevisionID: br.ID})
	if err != nil {
		logger.Printf(styles.Errorf("\n%s Failed to get promotion. Please check %s if a new revision was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, pushProjectID)))
		return nil
	}

	logger.Printf("\n%s Updating your Builder instance with the new revision...\n\n", emoji.Tools)

	readCloserPromotion, err := client.GetReleaseLogs(&api.GetReleaseLogsRequest{
		ID: p.ID,
	})
	if err != nil {
		logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		return nil
	}

	defer readCloserPromotion.Close()
	scannerPromotion := bufio.NewScanner(readCloserPromotion)
	for scannerPromotion.Scan() {
		// we don't want to print the logs to the terminal
	}
	if err := scannerPromotion.Err(); err != nil {
		logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return nil
	}

	// check promotion status
	p, err = client.GetReleasePromotion(&api.GetReleasePromotionRequest{PromotionID: p.ID})
	if err != nil {
		logger.Printf(styles.Errorf("\n%s Failed to check if Builder instance was updated. Please check %s", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, releaseProjectID)))
		return nil
	}
	if p.Status != api.Complete {
		logger.Println(styles.Errorf("\n%s Failed to update Builder instance. Please try again!", emoji.ErrorExclamation))
		return nil
	}

	// get installation via promotion id (promotion id == release id)
	i, err := client.GetInstallationByRelease(&api.GetInstallationByReleaseRequest{ReleaseID: p.ID})
	if err != nil {
		logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		logger.Printf(styles.Errorf("\n%s Failed to get installation. Please check %s if your Builder instance is being updated.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, pushProjectID)))
		return nil
	}

	readCloserInstallation, err := client.GetInstallationLogs(&api.GetInstallationLogsRequest{
		ID: i.ID,
	})
	if err != nil {
		logger.Println(styles.Errorf("%s Error: %v", emoji.ErrorExclamation, err))
		return nil
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
		logger.Printf("%s Error: %v\n", emoji.ErrorExclamation, err)
		return nil
	}

	// check installation status
	i, err = client.GetInstallation(&api.GetInstallationRequest{ID: i.ID})
	if err != nil {
		logger.Printf(styles.Errorf("\n%s Failed to check if Builder instance was updated. Please check %s", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, releaseProjectID)))
		return nil
	}
	if i.Status != api.Complete {
		logger.Println(styles.Errorf("\n%s Failed to update Builder instance. Please try again!", emoji.ErrorExclamation))
		return nil
	}

	logger.Println(styles.Greenf("\n%s Successfully pushed your code and updated your Builder instance!", emoji.PartyPopper))

	if instanceUrl != "" {
		logger.Printf("Builder instance: %s", styles.Code(instanceUrl))

		if pushOpen {
			err = browser.OpenURL(instanceUrl)

			if err != nil {
				return fmt.Errorf("%s Failed to open browser window %w", emoji.ErrorExclamation, err)
			}
		}
	}

	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", styles.Info, styles.Code("space version upgrade")))
	}
	return nil

}

package cmd

import (
	"bufio"
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/discovery"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/spinner"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
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
			return fmt.Errorf("problem while trying to get project id to push from prompt, %w", err)
		}
	}

	// check if spacefile is present
	isSpacefilePrsent, err := spacefile.IsSpacefilePresent(pushProjectDir)
	if err != nil {
		return err
	}
	if !isSpacefilePrsent {
		logger.Println(styles.Errorf("%s No Spacefile is present. Please add a Spacefile before pushing code.", emoji.ErrorExclamation))
		return nil
	}

	// parse spacefile and validate
	logger.Printf("Validating Spacefile...\n\n")

	s, err := spacefile.Open(projectDir)
	if err != nil {
		logger.Println(styles.Error(fmt.Sprintf("%s Error: %v", emoji.ErrorExclamation, err)))
		return nil
	}
	spacefileErrors := spacefile.ValidateSpacefile(s)

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
			r, err := client.CreateBuild(&api.CreateBuildRequest{AppID: pushProjectID})

			return spinner.Stop{
				RequestResponse: spinner.RequestResponse{Response: r, Err: err},
				FinishMsg:       fmt.Sprintf("%s Successfully started your build!", emoji.Check),
			}
		},
	}
	r := spinner.Run(&buildSpinnerInput)
	if r.Err != nil {
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
		logger.Println(styles.Errorf("\n%s Failed to push Spacefile.\n", emoji.ErrorExclamation))
		return r.Err
	}

	// push spacefile icon
	icon, err := s.GetIcon()
	if err != nil {
		logger.Println(styles.Errorf("\n%s Failed to get icon.\n", emoji.ErrorExclamation))
		return err
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
	r = spinner.Run(&pushSpacefileIcon)
	if r.Err != nil {
		logger.Println(styles.Errorf("\n%s Failed to push icon\n", emoji.ErrorExclamation))
		return r.Err
	}

	// push discovery file
	df, err := discovery.Open(pushProjectDir)
	if err != nil {
		logger.Println(styles.Errorf("\n%s Failed to read Discovery file.\n", emoji.ErrorExclamation))
		return err
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
	r = spinner.Run(&pushDiscoveryFile)
	if r.Err != nil {
		logger.Println(styles.Errorf("\n%s Failed to push Discovery file.\n", emoji.ErrorExclamation))
		return err
	}

	// push code & run build steps
	logger.Printf("%s Pushing your code & running build process...\n", emoji.Package)
	zippedCode, err := runtime.ZipDir(pushProjectDir)
	if err != nil {
		return err
	}
	if _, err = client.PushCode(&api.PushCodeRequest{
		BuildID: br.ID, ZippedCode: zippedCode,
	}); err != nil {
		return err
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
	b, err := client.GetBuild(&api.GetBuildLogsRequest{BuildID: br.ID})
	if err != nil {
		logger.Printf(styles.Errorf("\n%s Failed to check if push succeded. Please check %s if a new revision was created successfully.", emoji.ErrorExclamation, styles.Codef("%s/%s/develop", builderUrl, pushProjectID)))
		return nil
	}

	if b.Status == api.Complete {
		logger.Println(styles.Greenf("\n%s Successfully pushed your code and created a new Revision!\n", emoji.PartyPopper))
		logger.Printf("Run %s to create an installable Release for this Revision.\n", styles.Code("space release"))
		return nil
	} else {
		logger.Println(styles.Errorf("\n%s Failed to push code and create a revision. Please try again!", emoji.ErrorExclamation))
		return nil
	}

}

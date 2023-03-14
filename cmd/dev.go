package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	devCmd = &cobra.Command{
		Use:               "dev",
		PersistentPreRunE: createDataKeyIfNotExists,
	}
	devUpCmd = &cobra.Command{
		Use: "up",
	}

	devRunCmd = &cobra.Command{
		Use:  "run",
		RunE: devRun,
	}
	devProxyCmd = &cobra.Command{
		Use: "proxy",
	}
	devTriggerCmd = &cobra.Command{
		Use: "trigger",
	}
)

func init() {
	// dev up
	devCmd.AddCommand(devUpCmd)

	// dev run
	devCmd.AddCommand(devRunCmd)

	// dev proxy
	devCmd.AddCommand(devProxyCmd)

	// dev trigger
	devCmd.AddCommand(devTriggerCmd)

	// dev
	devCmd.PersistentFlags().StringP("dir", "d", "./", "directory of the Spacefile")
	devCmd.PersistentFlags().StringP("id", "i", "", "project id of the project to run")
	rootCmd.AddCommand(devCmd)
}

func createDataKeyIfNotExists(cmd *cobra.Command, args []string) error {
	projectDirectory, _ := cmd.Flags().GetString("dir")
	runtimeManager, err := runtime.NewManager(&projectDirectory, true)
	if err != nil {
		return err
	}

	isProjectInitialized, err := runtimeManager.IsProjectInitialized()
	if err != nil {
		return err
	}

	var devProjectID string
	// check if project is initialized
	if isProjectInitialized {
		projectMeta, err := runtimeManager.GetProjectMeta()
		if err != nil {
			return err
		}
		devProjectID = projectMeta.ID
		cmd.Flags().Set("id", devProjectID)
	} else if isFlagEmpty(devProjectID) {
		logger.Printf("No project was found in the current directory.\n\n")
		logger.Printf("Please provide using the space link command.\n\n")
		return errors.New("no project found")
	}

	// check if spacefile is present
	isSpacefilePresent, err := spacefile.IsSpacefilePresent(projectDirectory)
	if err != nil {
		if errors.Is(err, spacefile.ErrSpacefileWrongCase) {
			logger.Printf("%s The Spacefile must be called exactly 'Spacefile'.\n", emoji.ErrorExclamation)
			return nil
		}
		return err
	}
	if !isSpacefilePresent {
		logger.Println(styles.Errorf("%s No Spacefile is present. Please add a Spacefile.", emoji.ErrorExclamation))
		return nil
	}

	logger.Printf("Validating Spacefile...\n\n")

	// parse spacefile and validate
	s, err := spacefile.Open(projectDirectory)
	if err != nil {
		if te, ok := err.(*yaml.TypeError); ok {
			logger.Println(spacefile.ParseSpacefileUnmarshallTypeError(te))
			return nil
		}
		logger.Println(styles.Error(fmt.Sprintf("%s Error: %v", emoji.ErrorExclamation, err)))
		return nil
	}
	spacefileErrors := spacefile.ValidateSpacefile(s)

	if len(spacefileErrors) > 0 {
		logValidationErrors(s, spacefileErrors)
		logger.Println(styles.Error("\nPlease try to fix the issues with your Spacefile."))
		return nil
	} else {
		logValidationErrors(s, spacefileErrors)
		logger.Printf(styles.Green("\nYour Spacefile looks good!\n"))
	}

	// check if we have already stored the project key based on the project's id
	_, err = auth.GetProjectKey(devProjectID)
	if err != nil {
		if errors.Is(err, auth.ErrNoProjectKeyFound) {
			logger.Printf("%sNo project key found, generating new key...\n", emoji.Key)

			hostname, err := os.Hostname()
			if err != nil {
				hostname = ""
			}

			name := fmt.Sprintf("dev %s", hostname)[:20]

			// create a new project key using the api
			r, err := client.CreateProjectKey(devProjectID, &api.CreateProjectKeyRequest{
				Name: name,
			})
			if err != nil {
				return err
			}

			// store the project key locally
			err = auth.StoreProjectKey(devProjectID, r.Value)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		logger.Printf("%sUsing existing project key", emoji.Key)
	}

	return nil
}

func devRun(cmd *cobra.Command, args []string) error {
	commandName := args[0]
	var commandArgs []string
	if len(args) > 1 {
		commandArgs = args[1:]
	}

	projectId, _ := cmd.Flags().GetString("id")
	directory, _ := cmd.Flags().GetString("dir")
	projectKey, _ := auth.GetProjectKey(projectId)

	command := exec.Command(commandName, commandArgs...)
	command.Env = os.Environ()
	command.Env = append(command.Env, fmt.Sprintf("DETA_PROJECT_KEY=%s", projectKey))
	command.Dir = directory
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	return command.Run()
}

package cmd

import (
	"fmt"
	"os"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func newCmdBuilder() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "builder",
		Short:    "Interact with the builder",
		PostRunE: utils.CheckLatestVersion,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
	}

	cmd.AddCommand(newCmdEnv())
	return cmd
}

func newCmdEnv() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "env",
		Short:    "Interact with the env variables in the dev instance of your project",
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")
			if !cmd.Flags().Changed("id") {
				var err error
				projectID, err = runtime.GetProjectID(projectDir)
				if err != nil {
					return fmt.Errorf("failed to get project id: %w", err)
				}
			}

			set, _ := cmd.Flags().GetString("set")
			get, _ := cmd.Flags().GetString("get")
			micro, _ := cmd.Flags().GetString("micro")
			if cmd.Flags().Changed("set") && cmd.Flags().Changed("get") {
				return fmt.Errorf("Both `set` and `get` are used at the same time")
			}

			if cmd.Flags().Changed("get") {
				return cmdEnvGetFn(micro, get, projectID)
			} else if cmd.Flags().Changed("set") {
				return cmdEnvSetFn(micro, set, projectID)
			} else {
				return cmd.Usage()
			}
		},
	}

	cmd.Flags().StringP("get", "g", "", "file name to write the env variables")
	cmd.Flags().StringP("set", "s", ".env", "file name to read env variables from")
	cmd.Flags().StringP("micro", "m", "", "micro name to operate on")
	cmd.Flags().StringP("id", "i", "", "`project_id` of project")
	cmd.Flags().StringP("dir", "d", "./", "src of project")

	cmd.MarkFlagDirname("dir")

	return cmd
}

func cmdEnvGetFn(microName string, file string, projectID string) error {
	devInstance, err := utils.Client.GetDevAppInstance(projectID)
	if err != nil {
		return err
	}

	microPtr, err := cmdEnvGetMicro(microName, devInstance)
	if err != nil {
		return err
	}

	envMap := make(map[string]string)
	for _, env := range microPtr.Presets.Environment {
		envMap[env.Name] = env.Value
	}

	err = godotenv.Write(envMap, file)
	if err != nil {
		return fmt.Errorf("failed to write to `%s` env file: %w", file, err)
	}

	utils.Logger.Printf("%s Wrote %d environment variables from the micro `%s` to the file `%s`",
		emoji.Check.Emoji, len(envMap), microPtr.Name, file)

	return nil
}

func cmdEnvSetFn(microName string, file string, projectID string) error {
	devInstance, err := utils.Client.GetDevAppInstance(projectID)
	if err != nil {
		return err
	}

	microPtr, err := cmdEnvGetMicro(microName, devInstance)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read `%s` env file: %w", file, err)
	}

	envMap, err := godotenv.UnmarshalBytes(data)
	if err != nil {
		return fmt.Errorf("failed to parse `%s` env file: %w", file, err)
	}

	// update the values in-place
	counter := 0
	for _, env := range microPtr.Presets.Environment {
		if value, ok := envMap[env.Name]; ok && value != env.Value {
			env.Value = value
			counter++
		}
	}

	// no need to update any env variable
	if counter == 0 {
		utils.Logger.Printf("%s Found 0 environment variables that needs to be updated on the `%s` micro ",
			emoji.Check.Emoji, microPtr.Name)

		return nil
	}

	err = utils.Client.PatchDevAppInstancePresets(devInstance.ID, microPtr)
	if err != nil {
		return fmt.Errorf("Failed to patch the dev instance env presets: %s", err)
	}

	utils.Logger.Printf("%s Updated %d environment variables on the `%s` micro ",
		emoji.Check.Emoji, counter, microPtr.Name)

	return nil
}

func cmdEnvGetMicro(microName string, devInstance *api.AppInstance) (*api.AppInstanceMicro, error) {
	if len(devInstance.Micros) == 1 && (microName == "" || devInstance.Micros[0].Name == microName) {
		return devInstance.Micros[0], nil
	}

	var microPtr *api.AppInstanceMicro
	for _, micro := range devInstance.Micros {
		if micro.Name == microName {
			microPtr = micro
			break
		}
	}

	if microPtr != nil {
		return microPtr, nil
	}

	if microName == "" {
		return nil, fmt.Errorf("please provide a valid micro name with the `--micro` flag")
	} else {
		return nil, fmt.Errorf("micro '%s' not found in this project", microName)
	}
}

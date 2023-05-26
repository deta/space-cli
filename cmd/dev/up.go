package dev

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newCmdDevUp() *cobra.Command {
	devUpCmd := &cobra.Command{
		Short:    "Start a single micro for local development",
		Use:      "up <micro>",
		PreRunE:  shared.CheckAll(shared.CheckProjectInitialized("dir"), shared.CheckNotEmpty("id")),
		PostRunE: shared.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")
			port, _ := cmd.Flags().GetInt("port")
			open, _ := cmd.Flags().GetBool("open")

			if !cmd.Flags().Changed("id") {
				projectID, err = runtime.GetProjectID(projectDir)
				if err != nil {
					return err
				}

			}

			if !cmd.Flags().Changed("port") {
				port, err = GetFreePort(devDefaultPort + 1)
				if err != nil {
					shared.Logger.Printf("%s Failed to get free port: %s", emoji.ErrorExclamation, err)
					return err
				}
			}

			if err := devUp(projectDir, projectID, port, args[0], open); err != nil {
				return err
			}

			return nil
		},
	}

	devUpCmd.Flags().StringP("dir", "d", ".", "directory of the project")
	devUpCmd.Flags().StringP("id", "i", "", "project id")
	devUpCmd.Flags().IntP("port", "p", 0, "port to run the micro on")
	devUpCmd.Flags().Bool("open", false, "open the app in the browser")

	return devUpCmd
}

func devUp(projectDir string, projectId string, port int, microName string, open bool) (err error) {

	spacefile, err := spacefile.LoadSpacefile(projectDir)
	if err != nil {
		return err
	}

	projectKey, err := shared.GenerateDataKeyIfNotExists(projectId)
	if err != nil {
		shared.Logger.Printf("%s Error generating the project key", emoji.ErrorExclamation)
		return err
	}

	for _, micro := range spacefile.Micros {
		if micro.Name != microName {
			continue
		}

		portFile := filepath.Join(projectDir, ".space", "micros", fmt.Sprintf("%s.port", microName))
		if _, err := os.Stat(portFile); err == nil {
			microPort, _ := parsePort(portFile)
			if isPortActive(microPort) {
				shared.Logger.Printf("%s %s is already running on port %d", emoji.X, styles.Green(microName), microPort)
			}
		}

		writePortFile(portFile, port)

		command, err := MicroCommand(micro, projectDir, projectKey, port, context.Background())
		if err != nil {
			if errors.Is(err, errNoDevCommand) {
				shared.Logger.Printf("%s micro %s has no dev command\n", emoji.X, micro.Name)
				shared.Logger.Printf("See %s to get started\n", styles.Blue(spaceDevDocsURL))
				return err
			}
			return err
		}
		defer os.Remove(portFile)

		if err := command.Start(); err != nil {
			return fmt.Errorf("failed to start %s: %s", styles.Green(microName), err.Error())
		}

		// If we receive a SIGINT or SIGTERM, we want to send a SIGTERM to the child process
		go func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs
			shared.Logger.Printf("\n\nShutting down...\n\n")

			command.Process.Signal(syscall.SIGTERM)
		}()

		if open {
			browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))
		}

		microUrl := fmt.Sprintf("http://localhost:%d", port)
		shared.Logger.Printf("\n%s Micro %s running on %s", styles.Green("✔️"), styles.Green(microName), styles.Blue(microUrl))
		shared.Logger.Printf("\n%s Use %s to emulate the routing of your Space app\n\n", emoji.LightBulb, styles.Blue("space dev proxy"))

		command.Wait()
		return nil
	}

	shared.Logger.Printf("micro %s not found", microName)
	return fmt.Errorf("micro %s not found", microName)
}

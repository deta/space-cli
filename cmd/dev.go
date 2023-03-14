package cmd

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/proxy"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/spf13/cobra"
	"mvdan.cc/sh/v3/shell"
)

const (
	DEV_PORT = 3000
)

var (
	devCmd = &cobra.Command{
		Use:               "dev",
		Short:             "Run your app locally",
		PersistentPreRunE: createDataKeyIfNotExists,
	}
	devUpCmd = &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			microsDir := path.Join(dir, ".space", "micros")
			if _, err := os.Stat(microsDir); os.IsNotExist(err) {
				return os.MkdirAll(microsDir, os.ModePerm)
			}

			return nil
		},
		Use:  "up <micro>",
		RunE: devUp,
	}

	devRunCmd = &cobra.Command{
		Use:  "run",
		RunE: devRun,
	}
	devProxyCmd = &cobra.Command{
		Use:  "proxy",
		RunE: devProxy,
	}
	devTriggerCmd = &cobra.Command{
		Use: "trigger",
	}
)

func init() {
	// dev up
	devUpCmd.Flags().IntP("port", "p", 0, "port to run the micro on")
	devCmd.AddCommand(devUpCmd)

	// dev run
	devCmd.AddCommand(devRunCmd)

	// dev proxy
	devProxyCmd.Flags().IntP("port", "p", DEV_PORT, "port to run the proxy on")
	devCmd.AddCommand(devProxyCmd)

	// dev trigger
	devCmd.AddCommand(devTriggerCmd)

	// dev
	devCmd.PersistentFlags().StringP("dir", "d", ".", "directory of the Spacefile")
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

func devUp(cmd *cobra.Command, args []string) (err error) {
	microName := args[0]

	projectDir, _ := cmd.Flags().GetString("dir")
	projectId, _ := cmd.Flags().GetString("id")
	port, _ := cmd.Flags().GetInt("port")

	spacefile, _ := spacefile.Open(projectDir)
	projectKey, _ := auth.GetProjectKey(projectId)

	for _, micro := range spacefile.Micros {
		if micro.Name != microName {
			continue
		}
		devCommand := micro.Dev
		if cmd.Flags().Changed("command") {
			devCommand, _ = cmd.Flags().GetString("command")
		}

		environ := map[string]string{
			"PORT":                      fmt.Sprintf("%d", port),
			"DETA_PROJECT_KEY":          projectKey,
			"DETA_SPACE_APP_HOSTNAME":   fmt.Sprintf("localhost:%d", port),
			"DETA_SPACE_APP_MICRO_NAME": microName,
			"DETA_SPACE_APP_MICRO_TYPE": micro.Type(),
		}

		fields, err := shell.Fields(devCommand, func(s string) string {
			if env, ok := environ[s]; ok {
				return env
			}

			return os.Getenv(s)
		})
		if err != nil {
			return err
		}

		commandName := fields[0]
		var commandArgs []string
		if len(fields) > 0 {
			commandArgs = fields[1:]
		}

		command := exec.Command(commandName, commandArgs...)
		command.Env = os.Environ()
		for key, value := range environ {
			command.Env = append(command.Env, fmt.Sprintf("%s=%s", key, value))
		}
		command.Dir = path.Join(projectDir, micro.Src)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Stdin = os.Stdin

		portFile := path.Join(projectDir, ".space", "micros", fmt.Sprintf("%s.port", microName))
		if err := os.WriteFile(portFile, []byte(fmt.Sprintf("%d", port)), 0644); err != nil {
			return err
		}
		defer os.Remove(portFile)

		// If we receive a SIGINT or SIGTERM, we want to send a SIGTERM to the child process
		go func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs
			command.Process.Signal(syscall.SIGTERM)
		}()

		command.Run()
		return nil
	}

	return fmt.Errorf("micro %s not found", microName)
}

func devProxy(cmd *cobra.Command, args []string) error {
	directory, _ := cmd.Flags().GetString("dir")
	port, _ := cmd.Flags().GetInt("port")

	routeDir := path.Join(directory, ".space", "micros")
	spacefile, _ := spacefile.Open(directory)

	routes := make([]proxy.ProxyRoute, 0)
	for _, micro := range spacefile.Micros {
		portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		portBytes, err := os.ReadFile(portFile)
		if err != nil {
			return err
		}

		microPort, err := strconv.Atoi(string(portBytes))
		if err != nil {
			return err
		}

		target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", microPort))

		log.Println("proxying", micro.Prefix(), "to", target.String())
		routes = append(routes, proxy.ProxyRoute{
			Prefix: micro.Prefix(),
			Target: target,
		})
	}

	reverseProxy := proxy.NewReverseProxy(routes)
	log.Println("listening on", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), reverseProxy)

	return nil
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

package cmd

import (
	"context"
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
	"github.com/deta/pc-cli/shared"
	"github.com/phayes/freeport"
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
		RunE:              dev,
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
	devCmd.Flags().IntP("port", "p", DEV_PORT, "port to run the proxy on")
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
	if port == 0 {
		port, err = freeport.GetFreePort()
		if err != nil {
			return err
		}
	}

	spacefile, _ := spacefile.Open(projectDir)
	projectKey, _ := auth.GetProjectKey(projectId)

	for _, micro := range spacefile.Micros {
		if micro.Name != microName {
			continue
		}

		portFile := path.Join(projectDir, ".space", "micros", fmt.Sprintf("%s.port", microName))
		isRunning, err := isMicroRunning(portFile)
		if err != nil {
			return err
		}

		if isRunning {
			logger.Printf("%s %s is already running on port %d", emoji.Rocket, microName, port)
		}

		devCommand := micro.Dev
		if cmd.Flags().Changed("command") {
			devCommand, _ = cmd.Flags().GetString("command")
		}

		if err := os.WriteFile(portFile, []byte(fmt.Sprintf("%d", port)), 0644); err != nil {
			return err
		}

		command, err := microCommand(micro, devCommand, projectDir, projectKey, port)
		if err != nil {
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

	reverseProxy, err := proxyFromDir(spacefile.Micros, routeDir)
	if err != nil {
		return err
	}

	log.Println("listening on", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), reverseProxy)

	return nil
}

func dev(cmd *cobra.Command, args []string) error {
	projectDir, _ := cmd.Flags().GetString("dir")
	projectId, _ := cmd.Flags().GetString("id")
	port, _ := cmd.Flags().GetInt("port")

	routeDir := path.Join(projectDir, ".space", "micros")
	spacefile, _ := spacefile.Open(projectDir)
	projectKey, _ := auth.GetProjectKey(projectId)

	// Detect running micros
	var stoppedMicros []*shared.Micro
	for _, micro := range spacefile.Micros {
		portFile := path.Join(routeDir, micro.Name)
		if isRunning, _ := isMicroRunning(portFile); !isRunning {
			stoppedMicros = append(stoppedMicros, micro)
		}
	}

	freePorts, err := freeport.GetFreePorts(len(stoppedMicros))
	if err != nil {
		return err
	}

	commands := make([]*exec.Cmd, 0, len(stoppedMicros))
	// Start missing micros
	for i, micro := range stoppedMicros {
		command, err := microCommand(micro, "", projectDir, projectKey, freePorts[i])
		if err != nil {
			return err
		}
		commands = append(commands, command)

		portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		os.WriteFile(portFile, []byte(fmt.Sprintf("%d", freePorts[i])), 0644)
		defer os.Remove(portFile)
	}
	proxy, err := proxyFromDir(spacefile.Micros, routeDir)
	if err != nil {
		return err
	}
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: proxy,
	}

	for _, command := range commands {
		err := command.Start()
		// We should kill the other processes if one fails to start
		if err != nil {
			return err
		}
	}

	// If we receive a SIGINT or SIGTERM, we want to send a SIGTERM to the child process
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		for _, command := range commands {
			command.Process.Signal(syscall.SIGTERM)
		}

		server.Shutdown(context.Background())
	}()

	server.ListenAndServe()
	return nil
}

func proxyFromDir(micros []*shared.Micro, routeDir string) (*proxy.ReverseProxy, error) {
	routes := make([]proxy.ProxyRoute, 0)
	for _, micro := range micros {
		portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		portBytes, err := os.ReadFile(portFile)
		if err != nil {
			return nil, err
		}

		microPort, err := strconv.Atoi(string(portBytes))
		if err != nil {
			return nil, err
		}

		target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", microPort))

		log.Println("proxying", micro.Prefix(), "to", target.String())
		routes = append(routes, proxy.ProxyRoute{
			Prefix: micro.Prefix(),
			Target: target,
		})
	}

	return proxy.NewReverseProxy(routes), nil
}

func isMicroRunning(portFile string) (bool, error) {
	if _, err := os.Stat(portFile); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	// check if the port is already in use
	portStr, err := os.ReadFile(portFile)
	if err != nil {
		return false, err
	}

	port, err := strconv.Atoi(string(portStr))
	if err != nil {
		return false, err
	}

	if _, err = net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second); err == nil {
		return true, nil
	}

	return false, err
}

func microCommand(micro *shared.Micro, command string, directory, projectKey string, port int) (*exec.Cmd, error) {
	if command == "" {
		command = micro.Dev
	}

	environ := map[string]string{
		"PORT":                      fmt.Sprintf("%d", port),
		"DETA_PROJECT_KEY":          projectKey,
		"DETA_SPACE_APP_HOSTNAME":   fmt.Sprintf("localhost:%d", port),
		"DETA_SPACE_APP_MICRO_NAME": micro.Name,
		"DETA_SPACE_APP_MICRO_TYPE": micro.Type(),
	}

	fields, err := shell.Fields(command, func(s string) string {
		if env, ok := environ[s]; ok {
			return env
		}

		return os.Getenv(s)
	})
	if err != nil {
		return nil, err
	}

	commandName := fields[0]
	var commandArgs []string
	if len(fields) > 0 {
		commandArgs = fields[1:]
	}

	cmd := exec.Command(commandName, commandArgs...)
	cmd.Env = os.Environ()
	for key, value := range environ {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Dir = path.Join(directory, micro.Src)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, nil
}

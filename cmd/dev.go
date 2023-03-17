package cmd

import (
	"bytes"
	"context"
	"encoding/json"
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
	devDefaultPort = 3000
	actionEndpoint = "__space/v0/actions"
)

var (
	engineToDevCommand = map[string]string{
		shared.React:     "npm run start -- --port $PORT",
		shared.Vue:       "npm run dev -- --port $PORT",
		shared.Svelte:    "npm run dev -- --port $PORT",
		shared.Next:      "npm run dev -- --port $PORT",
		shared.Nuxt:      "npm run dev -- --port $PORT",
		shared.SvelteKit: "npm run dev -- --port $PORT",
	}
)

var (
	devCmd = &cobra.Command{
		Use:               "dev",
		Short:             "Run your app locally",
		PersistentPreRunE: devPreRun,
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
		Use:  "trigger",
		Args: cobra.ExactArgs(1),
		RunE: devTrigger,
	}
)

func init() {
	// dev up
	devUpCmd.Flags().IntP("port", "p", 0, "port to run the micro on")
	devCmd.AddCommand(devUpCmd)

	// dev run
	devCmd.AddCommand(devRunCmd)

	// dev proxy
	devProxyCmd.Flags().IntP("port", "p", devDefaultPort, "port to run the proxy on")
	devProxyCmd.Flags().StringP("host", "H", "localhost", "host to run the proxy on")
	devCmd.AddCommand(devProxyCmd)

	// dev trigger
	devCmd.AddCommand(devTriggerCmd)

	// dev
	devCmd.PersistentFlags().StringP("dir", "d", ".", "directory of the Spacefile")
	devCmd.PersistentFlags().StringP("id", "i", "", "project id of the project to run")
	devCmd.Flags().IntP("port", "p", devDefaultPort, "port to run the proxy on")
	devCmd.Flags().StringP("host", "H", "localhost", "host to run the proxy on")
	rootCmd.AddCommand(devCmd)
}

func devPreRun(cmd *cobra.Command, args []string) error {
	projectDirectory, _ := cmd.Flags().GetString("dir")

	devProjectID, _ := cmd.Flags().GetString("id")
	if devProjectID != "" {
		runtimeManager, err := runtime.NewManager(&projectDirectory, true)
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
			devProjectID = projectMeta.ID
			cmd.Flags().Set("id", devProjectID)
		} else if isFlagEmpty(devProjectID) {
			logger.Printf("No project was found in the current directory.\n\n")
			logger.Printf("Please provide using the space link command.\n\n")
			return errors.New("no project found")
		}
	}

	if err := generateDataKeyIfNeeded(devProjectID); err != nil {
		return err
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

	return nil
}

func findAvailableKey(res *api.ListProjectResponse, name string) string {
	keyMap := make(map[string]struct{})
	for _, key := range res.Keys {
		keyMap[key.Name] = struct{}{}
	}

	if _, ok := keyMap[name]; !ok {
		return name
	}

	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s (%d)", name, i)
		if _, ok := keyMap[newName]; !ok {
			return newName
		}
	}
}

func generateDataKeyIfNeeded(projectID string) error {
	// check if we have already stored the project key based on the project's id
	_, err := auth.GetProjectKey(projectID)
	if err == nil {
		return nil
	}

	if !errors.Is(err, auth.ErrNoProjectKeyFound) {
		return err
	}

	logger.Printf("%sNo project key found, generating new key...\n", emoji.Key)
	listRes, err := client.ListProjectKeys(projectID)
	if err != nil {
		return err
	}

	keyName := findAvailableKey(listRes, "space dev")

	// create a new project key using the api
	r, err := client.CreateProjectKey(projectID, &api.CreateProjectKeyRequest{
		Name: keyName,
	})
	if err != nil {
		return err
	}

	// store the project key locally
	err = auth.StoreProjectKey(projectID, r.Value)
	if err != nil {
		return err
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
		if _, err := os.Stat(portFile); err == nil {
			microPort, _ := parsePort(portFile)
			if isPortActive(microPort) {
				return fmt.Errorf("%s %s is already running on port %d", emoji.Rocket, microName, microPort)
			}
		}

		writePortFile(portFile, port)

		devCommand, _ := cmd.Flags().GetString("command")
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

type ActionEvent struct {
	ID      string `json:"id"`
	Trigger string `json:"trigger"`
}

type ActionRequest struct {
	Event ActionEvent `json:"event"`
}

func devTrigger(cmd *cobra.Command, args []string) (err error) {
	directory, _ := cmd.Flags().GetString("dir")
	spacefile, _ := spacefile.Open(directory)
	routeDir := path.Join(directory, ".space", "micros")

	for _, micros := range spacefile.Micros {
		for _, action := range micros.Actions {
			if action.ID != args[0] {
				continue
			}

			portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micros.Name))
			if _, err := os.Stat(portFile); err != nil {
				return fmt.Errorf("micro %s is not running", micros.Name)
			}

			port, err := parsePort(portFile)
			if err != nil {
				return err
			}

			if !isPortActive(port) {
				return fmt.Errorf("micro %s is not running", micros.Name)
			}

			body, err := json.Marshal(ActionRequest{
				Event: ActionEvent{
					ID:      args[0],
					Trigger: "schedule",
				},
			})
			if err != nil {
				return err
			}

			if _, err := http.Post(fmt.Sprintf("http://localhost:%d/%s", port, actionEndpoint), "application/json", bytes.NewReader(body)); err != nil {
				return err
			}
			return nil
		}
	}

	return errors.New("action not found")
}

func dev(cmd *cobra.Command, args []string) error {
	projectDir, _ := cmd.Flags().GetString("dir")
	projectId, _ := cmd.Flags().GetString("id")
	port, _ := cmd.Flags().GetInt("port")

	routeDir := path.Join(projectDir, ".space", "micros")
	spacefile, _ := spacefile.Open(projectDir)
	projectKey, _ := auth.GetProjectKey(projectId)

	var microT []*shared.Micro
	for _, micro := range spacefile.Micros {
		portFile := path.Join(routeDir, micro.Name)
		if _, err := os.Stat(portFile); err != nil {
			microT = append(microT, micro)
			continue
		}

		microPort, err := parsePort(portFile)
		if err != nil {
			microT = append(microT, micro)
			continue
		}

		if isRunning := isPortActive(microPort); !isRunning {
			microT = append(microT, micro)
			continue
		}
	}

	freePorts, err := freeport.GetFreePorts(len(microT))
	if err != nil {
		return err
	}

	commands := make([]*exec.Cmd, 0, len(microT))
	// Start missing micros
	for i, micro := range microT {
		command, err := microCommand(micro, "", projectDir, projectKey, freePorts[i])
		if err != nil {
			return err
		}
		commands = append(commands, command)

		portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		if err := writePortFile(portFile, freePorts[i]); err != nil {
			return err
		}
		defer os.Remove(portFile)
	}
	proxy, err := proxyFromDir(spacefile.Micros, routeDir)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("localhost:%d", port)
	server := http.Server{
		Addr:    addr,
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
		log.Println("shutting down")
		for _, command := range commands {
			command.Process.Signal(syscall.SIGTERM)
		}

		time.Sleep(1 * time.Second)

		server.Shutdown(context.Background())
	}()

	log.Println("listening on", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func writePortFile(filepath string, port int) error {
	portDir := path.Dir(filepath)
	if _, err := os.Stat(portDir); os.IsNotExist(err) {
		if err := os.MkdirAll(portDir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(filepath, []byte(fmt.Sprintf("%d", port)), 0644)
}

func proxyFromDir(micros []*shared.Micro, routeDir string) (*proxy.ReverseProxy, error) {
	routes := make([]proxy.ProxyRoute, 0)
	for _, micro := range micros {
		portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		if _, err := os.Stat(portFile); err != nil {
			continue
		}

		microPort, err := parsePort(portFile)
		if err != nil {
			continue
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

func parsePort(portFile string) (int, error) {
	// check if the port is already in use
	portStr, err := os.ReadFile(portFile)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(portStr))
}

func isPortActive(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}

	conn.Close()
	return true
}

func microCommand(micro *shared.Micro, command string, directory, projectKey string, port int) (*exec.Cmd, error) {
	var devCommand string
	if command != "" {
		devCommand = command
	} else if micro.Dev != "" {
		devCommand = micro.Dev
	} else if engineToDevCommand[micro.Engine] != "" {
		devCommand = engineToDevCommand[micro.Engine]
	} else {
		return nil, fmt.Errorf("no dev command found for micro %s", micro.Name)
	}

	commandDir := directory
	if micro.Src != "" {
		commandDir = path.Join(directory, micro.Src)
	} else {
		commandDir = micro.Name
	}

	environ := map[string]string{
		"PORT":                      fmt.Sprintf("%d", port),
		"DETA_PROJECT_KEY":          projectKey,
		"DETA_SPACE_APP_HOSTNAME":   fmt.Sprintf("localhost:%d", port),
		"DETA_SPACE_APP_MICRO_NAME": micro.Name,
		"DETA_SPACE_APP_MICRO_TYPE": micro.Type(),
	}

	fields, err := shell.Fields(devCommand, func(s string) string {
		if env, ok := environ[s]; ok {
			return env
		}

		return os.Getenv(s)
	})
	if err != nil {
		return nil, err
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("no command found for micro %s", micro.Name)
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
	cmd.Dir = commandDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, nil
}

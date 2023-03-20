package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
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
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"mvdan.cc/sh/v3/shell"
)

const (
	devDefaultPort = 4200
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
		Short:             "Spin up a local development environment for your Space project",
		PersistentPreRunE: devPreRun,
		RunE:              dev,
	}
	devUpCmd = &cobra.Command{
		Short: "Start a local server for a specific micro",
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
		Use:   "run <command>",
		Short: "Run a command in the project's environment",
		Args:  cobra.MinimumNArgs(1),
		RunE:  devRun,
	}
	devProxyCmd = &cobra.Command{
		Use:   "proxy",
		Short: "Start the proxy server for your running micros",
		RunE:  devProxy,
	}
	devTriggerCmd = &cobra.Command{
		Use:   "trigger <action>",
		Short: "Trigger a micro action",
		Args:  cobra.ExactArgs(1),
		RunE:  devTrigger,
	}
)

func init() {
	// dev up
	devUpCmd.Flags().IntP("port", "p", 0, "port to run the micro on")
	devUpCmd.Flags().Bool("open", false, "open the app in the browser")
	devCmd.AddCommand(devUpCmd)

	// dev run
	devCmd.AddCommand(devRunCmd)

	// dev proxy
	devProxyCmd.Flags().IntP("port", "p", devDefaultPort, "port to run the proxy on")
	devProxyCmd.Flags().StringP("host", "H", "localhost", "host to run the proxy on")
	devProxyCmd.Flags().Bool("open", false, "open the app in the browser")
	devCmd.AddCommand(devProxyCmd)

	// dev trigger
	devCmd.AddCommand(devTriggerCmd)

	// dev
	devCmd.PersistentFlags().StringP("dir", "d", ".", "directory of the Spacefile")
	devCmd.PersistentFlags().StringP("id", "i", "", "project id of the project to run")
	devCmd.Flags().IntP("port", "p", devDefaultPort, "port to run the proxy on")
	devCmd.Flags().StringP("host", "H", "localhost", "host to run the proxy on")
	devCmd.Flags().Bool("open", false, "open the app in the browser")
	rootCmd.AddCommand(devCmd)
}

func devPreRun(cmd *cobra.Command, args []string) error {
	projectDirectory, _ := cmd.Flags().GetString("dir")

	var devProjectID string
	if !cmd.Flags().Changed("id") {
		metaPath := path.Join(projectDirectory, ".space", "meta")
		bytes, err := os.ReadFile(metaPath)
		if err != nil {
			logger.Printf("%sCould not read project meta file. Please run `deta new` to create a new project.\n", emoji.X)
			os.Exit(1)
		}

		var meta runtime.ProjectMeta
		if err := json.Unmarshal(bytes, &meta); err != nil {
			return err
		}

		devProjectID = meta.ID
		cmd.Flags().Set("id", devProjectID)

	} else {
		devProjectID, _ = cmd.Flags().GetString("id")
	}

	// parse spacefile and validate
	s, err := spacefile.Open(projectDirectory)
	if err != nil {
		if te, ok := err.(*yaml.TypeError); ok {
			logger.Println(spacefile.ParseSpacefileUnmarshallTypeError(te))
			return nil
		}
		logger.Println(styles.Error(fmt.Sprintf("%sError: %v", emoji.ErrorExclamation, err)))
		return nil
	}

	logger.Printf("%sValidating Spacefile...", emoji.V)
	if spacefileErrors := spacefile.ValidateSpacefile(s, projectDirectory); len(spacefileErrors) > 0 {
		logValidationErrors(s, spacefileErrors)
		logger.Println("Please fix the errors in your Spacefile and try again.")
		os.Exit(1)
	}

	if _, err := auth.GetProjectKey(devProjectID); err != nil {
		logger.Printf("%sGenerating new project key...\n", emoji.Key)
		err := generateDataKey(devProjectID)
		if err != nil {
			return err
		}
	} else {
		logger.Printf("%sUsing existing project key...\n", emoji.Key)
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

func generateDataKey(projectID string) error {
	// check if we have already stored the project key based on the project's id
	logger.Printf("%s No project key found, generating new key...\n", emoji.Key)
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

	logger.Printf("\n┌ Command output:\n\n")

	return command.Run()
}

func GetFreePort(start int) (int, error) {
	if start < 0 || start > 65535 {
		return 0, errors.New("invalid port range")
	}

	for portNumber := start; portNumber < start+100; portNumber++ {
		if isPortActive(portNumber) {
			continue
		}

		return portNumber, nil
	}

	return 0, errors.New("no free port found")
}

func devUp(cmd *cobra.Command, args []string) (err error) {
	microName := args[0]

	projectDir, _ := cmd.Flags().GetString("dir")
	projectId, _ := cmd.Flags().GetString("id")

	var port int
	if cmd.Flags().Changed("port") {
		port, _ = cmd.Flags().GetInt("port")
	} else {
		port, err = GetFreePort(devDefaultPort + 1)
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
				logger.Printf("%s%s is already running on port %d", emoji.X, styles.Green(microName), microPort)
			}
		}

		writePortFile(portFile, port)

		devCommand, _ := cmd.Flags().GetString("command")
		command, err := microCommand(micro, devCommand, projectDir, projectKey, port)
		if err != nil {
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
			command.Process.Signal(syscall.SIGTERM)
		}()

		if open, _ := cmd.Flags().GetBool("open"); open {
			browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))
		}

		microUrl := fmt.Sprintf("http://localhost:%d", port)
		logger.Printf("\n%sMicro %s listening on %s\n\n", emoji.V, styles.Green(microName), styles.Blue(microUrl))

		command.Wait()
		return nil
	}

	return fmt.Errorf("micro %s not found", microName)
}

func devProxy(cmd *cobra.Command, args []string) error {
	directory, _ := cmd.Flags().GetString("dir")
	host, _ := cmd.Flags().GetString("host")

	var port int
	if cmd.Flags().Changed("port") {
		port, _ = cmd.Flags().GetInt("port")
	} else {
		port, _ = GetFreePort(devDefaultPort)
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	microDir := path.Join(directory, ".space", "micros")
	spacefile, _ := spacefile.Open(directory)

	if entries, err := os.ReadDir(microDir); err != nil || len(entries) == 0 {
		logger.Printf("\n%sNo running micros detected.", emoji.X)
		logger.Printf("L Use %s to manually start a micro", styles.Blue("space dev up <micro>"))
		os.Exit(1)
	}

	reverseProxy, err := proxyFromDir(spacefile.Micros, microDir)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:    addr,
		Handler: reverseProxy,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Printf("%sproxy listening on http://%s", emoji.Laptop, addr)
		server.ListenAndServe()
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		server.Shutdown(context.Background())
	}()

	if open, _ := cmd.Flags().GetBool("open"); open {
		browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))
	}

	wg.Wait()
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

	actionId := args[0]

	for _, micro := range spacefile.Micros {
		for _, action := range micro.Actions {
			if action.ID != actionId {
				continue
			}

			logger.Printf("\n%sChecking if micro %s is running...\n", emoji.Laptop, styles.Green(micro.Name))
			port, err := getMicroPort(micro, routeDir)
			if err != nil {
				upCommand := fmt.Sprintf("space dev up %s", micro.Name)
				logger.Printf("%smicro %s is not running, to start it run:", emoji.X, styles.Green(micro.Name))
				logger.Printf("L %s", styles.Blue(upCommand))
				os.Exit(1)
			}

			logger.Printf("%sMicro %s is running", emoji.V, styles.Green(micro.Name))

			body, err := json.Marshal(ActionRequest{
				Event: ActionEvent{
					ID:      actionId,
					Trigger: "schedule",
				},
			})
			if err != nil {
				return err
			}

			actionEndpoint := fmt.Sprintf("http://localhost:%d/%s", port, actionEndpoint)
			logger.Printf("\nTriggering action %s", styles.Green(actionId))
			logger.Printf("L POST %s", styles.Blue(actionEndpoint))

			res, err := http.Post(actionEndpoint, "application/json", bytes.NewReader(body))
			if err != nil {
				logger.Printf("\n%sfailed to trigger action: %s", emoji.X, err.Error())
				os.Exit(1)
			}
			defer res.Body.Close()

			logger.Println("\n┌ Action Response:")

			logger.Printf("\n%s", res.Status)

			logger.Println()
			io.Copy(os.Stdout, res.Body)

			if res.StatusCode >= 400 {
				logger.Printf("\n\nL %s", styles.Error("failed to trigger action"))
				os.Exit(1)
			}
			logger.Printf("\n\nL Action triggered successfully!")
			return nil
		}
	}

	logger.Printf("\n%saction `%s` not found", emoji.X, actionId)
	os.Exit(1)
	return nil
}

func dev(cmd *cobra.Command, args []string) error {
	projectDir, _ := cmd.Flags().GetString("dir")
	projectId, _ := cmd.Flags().GetString("id")
	host, _ := cmd.Flags().GetString("host")

	routeDir := path.Join(projectDir, ".space", "micros")
	spacefile, _ := spacefile.Open(projectDir)
	projectKey, _ := auth.GetProjectKey(projectId)

	var proxyPort int
	if cmd.Flags().Changed("port") {
		proxyPort, _ = cmd.Flags().GetInt("port")
	} else {
		var err error
		proxyPort, err = GetFreePort(devDefaultPort)
		if err != nil {
			return err
		}
	}
	addr := fmt.Sprintf("%s:%d", host, proxyPort)

	var stoppedMicros []*shared.Micro
	logger.Printf("\n%s Checking for running micros...\n", emoji.Laptop)
	for _, micro := range spacefile.Micros {
		port, err := getMicroPort(micro, routeDir)
		if err != nil {
			logger.Printf("micro %s is not running\n", styles.Green(micro.Name))
			stoppedMicros = append(stoppedMicros, micro)
			continue
		}

		logger.Printf("%s micro %s is already running on port %d\n\n", emoji.V, micro.Name, port)
	}

	commands := make([]*exec.Cmd, 0, len(stoppedMicros))
	startPort := proxyPort + 1

	logger.Printf("\n%s Starting %d micro servers...\n", emoji.Laptop, len(stoppedMicros))
	for _, micro := range stoppedMicros {
		freePort, err := GetFreePort(startPort)
		if err != nil {
			return err
		}

		command, err := microCommand(micro, "", projectDir, projectKey, freePort)
		if err != nil {
			return err
		}

		portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		if err := writePortFile(portFile, freePort); err != nil {
			return err
		}
		defer os.Remove(portFile)

		if micro.Primary {
			logger.Printf("\nMicro %s (primary)", styles.Green(micro.Name))
		} else {
			logger.Printf("\nMicro %s", styles.Green(micro.Name))
		}

		logger.Printf("L command: %s\n", styles.Blue(command.String()))
		spaceUrl := fmt.Sprintf("http://%s%s", addr, micro.Route())
		logger.Printf("L url: %s\n", styles.Blue(spaceUrl))

		commands = append(commands, command)
		startPort = freePort + 1
	}

	proxy, err := proxyFromDir(spacefile.Micros, routeDir)
	if err != nil {
		return err
	}

	server := http.Server{
		Addr:    addr,
		Handler: proxy,
	}

	wg := sync.WaitGroup{}

	for _, command := range commands {
		wg.Add(1)
		go func(command *exec.Cmd) {
			defer wg.Done()
			command.Run()
		}(command)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		appUrl := fmt.Sprintf("http://%s", addr)
		logger.Printf("\n%sApp available at %s\n\n", emoji.Rocket, styles.Blue(appUrl))
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Println("proxy error", err)
		}
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		logger.Println("\nShutting down all commands...")
		for _, command := range commands {
			command.Process.Signal(syscall.SIGTERM)
		}
		server.Shutdown(context.Background())
	}()

	if open, _ := cmd.Flags().GetBool("open"); open {
		browser.OpenURL(fmt.Sprintf("http://%s", addr))
	}

	wg.Wait()

	// Wait a bit for all logs to be printed
	time.Sleep(1 * time.Second)

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

		routes = append(routes, proxy.ProxyRoute{
			Prefix: micro.Route(),
			Target: target,
		})
	}

	return proxy.NewReverseProxy(routes), nil
}

func getMicroPort(micro *shared.Micro, routeDir string) (int, error) {
	portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
	if _, err := os.Stat(portFile); err != nil {
		return 0, err
	}

	port, err := parsePort(portFile)
	if err != nil {
		return 0, err
	}

	if !isPortActive(port) {
		return 0, fmt.Errorf("port %d is not active", port)
	}

	return port, nil
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
		commandDir = path.Join(directory, micro.Name)
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
	cmd.Stdout = NewPrefixer(micro.Name, os.Stdout)
	cmd.Stderr = NewPrefixer(micro.Name, os.Stderr)

	return cmd, nil
}

type Prefixer struct {
	scope string
	dest  io.Writer
}

func NewPrefixer(scope string, dest io.Writer) *Prefixer {
	return &Prefixer{
		scope: scope,
		dest:  dest,
	}
}

// parse the logs and prefix them with the scope
func (p Prefixer) Write(bytes []byte) (int, error) {
	normalized := strings.ReplaceAll(string(bytes), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	for _, line := range lines {
		fmt.Printf("[%s] %s\n", p.scope, line)
	}

	return len(bytes), nil
}

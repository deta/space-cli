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
)

const (
	devDefaultPort  = 4200
	actionEndpoint  = "__space/v0/actions"
	spaceDevDocsURL = "https://deta.space/docs/en/basics/local"
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

	devProxyCmd = &cobra.Command{
		Use:   "proxy",
		Short: "Start the proxy server for your running micros",
		RunE:  devProxy,
	}
	devTriggerCmd = &cobra.Command{
		Use:     "trigger <action>",
		Short:   "Trigger a micro action",
		Aliases: []string{"t"},
		Args:    cobra.ExactArgs(1),
		RunE:    devTrigger,
	}
)

func init() {
	// dev up
	devUpCmd.Flags().IntP("port", "p", 0, "port to run the micro on")
	devUpCmd.Flags().Bool("open", false, "open the app in the browser")
	devCmd.AddCommand(devUpCmd)

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
			logger.Printf("%sCould not read project metadatas.", emoji.X)
			logger.Printf("L Run `%s` in the project directory to create a new project or `%s` to link a existing one.", styles.Blue("space new"), styles.Blue("space link"))
			os.Exit(1)
		}

		var meta runtime.ProjectMeta
		if err := json.Unmarshal(bytes, &meta); err != nil {
			return err
		}

		devProjectID = meta.ID
		cmd.Flags().Set("id", devProjectID)

	}

	// parse spacefile and validate
	s, err := spacefile.Open(projectDirectory)
	if err != nil {
		if te, ok := err.(*yaml.TypeError); ok {
			logger.Println(spacefile.ParseSpacefileUnmarshallTypeError(te))
			os.Exit(1)
		}
		logger.Println(styles.Error(fmt.Sprintf("%sError: %v", emoji.ErrorExclamation, err)))
		os.Exit(1)
	}

	logger.Printf("%s Validating Spacefile...", styles.Green("✔️"))
	if spacefileErrors := spacefile.ValidateSpacefile(s, projectDirectory); len(spacefileErrors) > 0 {
		logValidationErrors(s, spacefileErrors)
		logger.Println("Please fix the errors in your Spacefile and try again.")
		os.Exit(1)
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

func generateDataKeyIfNotExists(projectID string) (string, error) {
	// check if we have already stored the project key based on the project's id
	projectKey, err := auth.GetProjectKey(projectID)
	if err == nil {
		logger.Printf("%sFound existing data key locally.", emoji.Key)
		return projectKey, nil
	}

	logger.Printf("%sGenerating new data key...", emoji.Key)
	listRes, err := client.ListProjectKeys(projectID)
	if err != nil {
		return "", err
	}

	keyName := findAvailableKey(listRes, "space dev")

	// create a new project key using the api
	r, err := client.CreateProjectKey(projectID, &api.CreateProjectKeyRequest{
		Name: keyName,
	})
	if err != nil {
		return "", err
	}

	// store the project key locally
	err = auth.StoreProjectKey(projectID, r.Value)
	if err != nil {
		return "", err
	}

	return r.Value, nil
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

	spacefile, _ := spacefile.Open(projectDir)
	projectKey, err := generateDataKeyIfNotExists(projectId)
	if err != nil {
		logger.Printf("%s Error generating the project key", emoji.ErrorExclamation)
		os.Exit(1)
	}

	var port int
	if cmd.Flags().Changed("port") {
		port, _ = cmd.Flags().GetInt("port")
	} else {
		port, err = GetFreePort(devDefaultPort + 1)
		if err != nil {
			return err
		}
	}

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

		command, err := micro.Command(projectDir, projectKey, port)
		if err != nil {
			if errors.Is(err, shared.ErrNoDevCommand) {
				logger.Printf("%s micro %s has no dev command\n", emoji.X, micro.Name)
				logger.Printf("See %s to get started\n", styles.Blue(spaceDevDocsURL))
				os.Exit(1)
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
			logger.Printf("\n\nShutting down...\n\n")

			command.Process.Signal(syscall.SIGTERM)
		}()

		if open, _ := cmd.Flags().GetBool("open"); open {
			browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))
		}

		microUrl := fmt.Sprintf("http://localhost:%d", port)
		logger.Printf("\n%s Micro %s running on %s", styles.Green("✔️"), styles.Green(microName), styles.Blue(microUrl))
		logger.Printf("\n%sUse %s to emulate the routing of your Space app\n\n", emoji.LightBulb, styles.Blue("space dev proxy"))

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
		logger.Printf("\n\nShutting down...\n\n")
		server.Shutdown(context.Background())
	}()

	if open, _ := cmd.Flags().GetBool("open"); open {
		browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))
	}

	wg.Wait()
	return nil
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

			logger.Printf("\n%sChecking if micro %s is running...\n", emoji.Eyes, styles.Green(micro.Name))
			port, err := getMicroPort(micro, routeDir)
			if err != nil {
				upCommand := fmt.Sprintf("space dev up %s", micro.Name)
				logger.Printf("%smicro %s is not running, to start it run:", emoji.X, styles.Green(micro.Name))
				logger.Printf("L %s", styles.Blue(upCommand))
				os.Exit(1)
			}

			logger.Printf("%s Micro %s is running", styles.Green("✔️"), styles.Green(micro.Name))

			body, err := json.Marshal(shared.ActionRequest{
				Event: shared.ActionEvent{
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
	projectKey, err := generateDataKeyIfNotExists(projectId)
	if err != nil {
		logger.Printf("%s Error generating the project key", emoji.ErrorExclamation)
		os.Exit(1)
	}

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

	logger.Printf("\n%sChecking for running micros...", emoji.Eyes)
	var stoppedMicros []*shared.Micro
	for _, micro := range spacefile.Micros {
		_, err := getMicroPort(micro, routeDir)
		if err != nil {
			stoppedMicros = append(stoppedMicros, micro)
			continue
		}

		logger.Printf("\nMicro %s found", styles.Green(micro.Name))
		logger.Printf("L url: %s", styles.Blue(fmt.Sprintf("http://%s%s", addr, micro.Path)))
	}

	commands := make([]*exec.Cmd, 0, len(stoppedMicros))
	startPort := proxyPort + 1

	logger.Printf("\n%sStarting %d micro servers...\n\n", emoji.Laptop, len(stoppedMicros))
	for _, micro := range stoppedMicros {
		freePort, err := GetFreePort(startPort)
		if err != nil {
			return err
		}

		command, err := micro.Command(projectDir, projectKey, freePort)
		if err != nil {
			if errors.Is(err, shared.ErrNoDevCommand) {
				logger.Printf("%s micro %s has no dev command\n", emoji.X, micro.Name)
				logger.Printf("See %s to get started\n", styles.Blue(spaceDevDocsURL))
				continue
			}
		}

		portFile := path.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		if err := writePortFile(portFile, freePort); err != nil {
			return err
		}
		defer os.Remove(portFile)

		commands = append(commands, command)
		startPort = freePort + 1

		if micro.Primary {
			logger.Printf("Micro %s (primary)", styles.Green(micro.Name))
		} else {
			logger.Printf("Micro %s", styles.Green(micro.Name))
		}
		spaceUrl := fmt.Sprintf("http://%s%s", addr, micro.Path)
		logger.Printf("L url: %s\n\n", styles.Blue(spaceUrl))
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
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Println("proxy error", err)
		}
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		logger.Printf("\n\nShutting down...\n\n")

		for _, command := range commands {
			command.Process.Signal(syscall.SIGTERM)
		}
		server.Shutdown(context.Background())
	}()

	if open, _ := cmd.Flags().GetBool("open"); open {
		// Wait a bit for the server to start
		time.Sleep(1 * time.Second)
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
			Prefix: micro.Path,
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

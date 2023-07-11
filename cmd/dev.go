package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/alessio/shellescape"
	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/proxy"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/deta/space/pkg/writer"
	types "github.com/deta/space/shared"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"mvdan.cc/sh/v3/shell"
)

const (
	actionEndpoint  = "__space/v0/actions"
	spaceDevDocsURL = "https://deta.space/docs/en/build/fundamentals/development/local-development"
)

var (
	EngineToDevCommand = map[string]string{
		types.React:     "npm run start -- --port $PORT",
		types.Vue:       "npm run dev -- --port $PORT",
		types.Svelte:    "npm run dev -- --port $PORT",
		types.Next:      "npm run dev -- --port $PORT",
		types.Nuxt:      "npm run dev -- --port $PORT",
		types.SvelteKit: "npm run dev -- --port $PORT",
	}
	errNoDevCommand = errors.New("no dev command found for micro")
)

func NewCmdDev() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Spin up a local development environment for your Space project",
		Long: `Spin up a local development environment for your Space project.

The cli will start one process for each of your micros, then expose a single enpoint for your Space app.`,

		PreRunE:  utils.CheckAll(utils.CheckProjectInitialized("dir"), utils.CheckNotEmpty("id")),
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			projectDir, _ := cmd.Flags().GetString("dir")
			projectID, _ := cmd.Flags().GetString("id")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			open, _ := cmd.Flags().GetBool("open")

			if !cmd.Flags().Changed("id") {
				projectID, err = runtime.GetProjectID(projectDir)
				if err != nil {
					utils.Logger.Printf("%s Failed to get project id: %s", emoji.ErrorExclamation, err)
					return err
				}
			}

			if !cmd.Flags().Changed("port") {
				port, err = GetFreePort(utils.DevPort)
				if err != nil {
					return err
				}
			}

			if err := dev(projectDir, projectID, host, port, open); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.AddCommand(newCmdDevUp())
	cmd.AddCommand(newCmdDevProxy())
	cmd.AddCommand(newCmdDevTrigger())
	cmd.AddCommand(newCmdServe())

	cmd.Flags().StringP("dir", "d", ".", "directory of the project")
	cmd.Flags().StringP("id", "i", "", "project id")
	cmd.Flags().IntP("port", "p", 0, "port to run the proxy on")
	cmd.Flags().StringP("host", "H", "localhost", "host to run the proxy on")
	cmd.Flags().Bool("open", false, "open the app in the browser")

	return cmd
}

func GetFreePort(start int) (int, error) {
	if start < 0 || start > 65535 {
		return 0, errors.New("invalid port range")
	}

	for portNumber := start; portNumber < start+100; portNumber++ {
		if utils.IsPortActive(portNumber) {
			continue
		}

		return portNumber, nil
	}

	return 0, errors.New("no free port found")
}

func dev(projectDir string, projectID string, host string, port int, open bool) error {
	meta, err := runtime.GetProjectMeta(projectDir)
	if err != nil {
		return err
	}

	routeDir := filepath.Join(projectDir, ".space", "micros")
	spacefile, err := spacefile.LoadSpacefile(projectDir)
	if err != nil {
		utils.Logger.Printf("%s Failed to parse Spacefile: %s", emoji.ErrorExclamation, err)
		return err
	}

	projectKey, err := utils.GenerateDataKeyIfNotExists(projectID)

	addr := fmt.Sprintf("%s:%d", host, port)
	if err != nil {
		utils.Logger.Printf("%s Error generating the project key", emoji.ErrorExclamation)
		return err
	}

	utils.Logger.Printf("\n%s Checking for running micros...", emoji.Eyes)
	var stoppedMicros []*types.Micro
	for _, micro := range spacefile.Micros {
		_, err := getMicroPort(micro, routeDir)
		if err != nil {
			stoppedMicros = append(stoppedMicros, micro)
			continue
		}

		utils.Logger.Printf("\nMicro %s found", styles.Green(micro.Name))
		utils.Logger.Printf("L url: %s", styles.Blue(fmt.Sprintf("http://%s%s", addr, micro.Path)))
	}

	startPort := port + 1

	wg := sync.WaitGroup{}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	utils.Logger.Printf("\n%s Starting %d micro servers...\n\n", emoji.Laptop, len(stoppedMicros))
	for _, micro := range stoppedMicros {
		freePort, err := GetFreePort(startPort)
		if err != nil {
			return err
		}

		command, err := MicroCommand(micro, projectDir, projectKey, freePort, ctx)
		if err != nil {
			if errors.Is(err, errNoDevCommand) {
				utils.Logger.Printf("%s micro %s has no dev command\n", emoji.X, micro.Name)
				utils.Logger.Printf("See %s to get started\n", styles.Blue(spaceDevDocsURL))
				continue
			}
		}

		portFile := filepath.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		if err := writePortFile(portFile, freePort); err != nil {
			return err
		}
		defer os.Remove(portFile)

		startPort = freePort + 1

		if micro.Primary {
			utils.Logger.Printf("Micro %s (primary)", styles.Green(micro.Name))
		} else {
			utils.Logger.Printf("Micro %s", styles.Green(micro.Name))
		}
		spaceUrl := fmt.Sprintf("http://%s%s", addr, micro.Path)
		utils.Logger.Printf("L url: %s\n\n", styles.Blue(spaceUrl))

		wg.Add(1)
		go func(command *exec.Cmd) {
			defer wg.Done()
			err := command.Run()
			if err != nil {
				if errors.Is(err, exec.ErrNotFound) {
					utils.Logger.Printf("%s Command not found: %s", emoji.ErrorExclamation, command.Args[0])
					return
				}
				utils.Logger.Printf("Command `%s` exited.", command.String())
				cancelFunc()
			}
		}(command)
	}

	time.Sleep(3 * time.Second)
	proxy := proxy.NewReverseProxy(meta.ID, meta.Name, meta.Alias)
	if err := loadMicrosFromDir(proxy, spacefile.Micros, routeDir); err != nil {
		return err
	}

	server := http.Server{
		Addr:    addr,
		Handler: proxy,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			utils.Logger.Println("proxy error", err)
		}
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-sigs:
			utils.Logger.Println("Interrupted!")
			cancelFunc()
		case <-ctx.Done():
		}

		utils.Logger.Printf("\n\nShutting down...\n\n")
		server.Shutdown(context.Background())
	}()

	if open {
		// Wait a bit for the server to start
		time.Sleep(1 * time.Second)
		browser.OpenURL(fmt.Sprintf("http://%s", addr))
	}

	wg.Wait()

	// Wait a bit for all logs to be printed
	time.Sleep(1 * time.Second)

	return nil
}

func writePortFile(portfile string, port int) error {
	portDir := filepath.Dir(portfile)
	if _, err := os.Stat(portDir); os.IsNotExist(err) {
		if err := os.MkdirAll(portDir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(portfile, []byte(fmt.Sprintf("%d", port)), 0644)
}

func loadMicrosFromDir(proxy *proxy.ReverseProxy, micros []*types.Micro, routeDir string) error {
	for _, micro := range micros {
		portFile := filepath.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
		if _, err := os.Stat(portFile); err != nil {
			continue
		}

		microPort, err := parsePort(portFile)
		if err != nil {
			continue
		}

		n, err := proxy.AddMicro(micro, microPort)
		if err != nil {
			return err
		}

		if n != 0 {
			utils.Logger.Printf("\nExtracted %d actions from %s.", n, micro.Name)
			utils.Logger.Printf("L Preview URL: %s\n\n", "https://deta.space?devServer=http://localhost:4200")
		}
	}

	return nil
}

func getMicroPort(micro *types.Micro, routeDir string) (int, error) {
	portFile := filepath.Join(routeDir, fmt.Sprintf("%s.port", micro.Name))
	if _, err := os.Stat(portFile); err != nil {
		return 0, err
	}

	port, err := parsePort(portFile)
	if err != nil {
		return 0, err
	}

	if !utils.IsPortActive(port) {
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

func MicroCommand(micro *types.Micro, directory, projectKey string, port int, ctx context.Context) (*exec.Cmd, error) {
	var devCommand string

	if micro.Dev != "" {
		devCommand = micro.Dev
	} else if micro.Engine == "static" {
		root := micro.Serve
		if root == "" {
			root = micro.Src
		}
		devCommand = fmt.Sprintf("%s dev serve %s --port %d", shellescape.Quote(os.Args[0]), shellescape.Quote(root), port)
	} else if EngineToDevCommand[micro.Engine] != "" {
		devCommand = EngineToDevCommand[micro.Engine]
	} else {
		return nil, errNoDevCommand
	}

	commandDir := filepath.Join(directory, micro.Src)

	environ := map[string]string{
		"PORT":                      fmt.Sprintf("%d", port),
		"DETA_PROJECT_KEY":          projectKey,
		"DETA_SPACE_APP_HOSTNAME":   fmt.Sprintf("localhost:%d", port),
		"DETA_SPACE_APP_MICRO_NAME": micro.Name,
		"DETA_SPACE_APP_MICRO_TYPE": micro.Type(),
	}

	if types.IsPythonEngine(micro.Engine) {
		environ["UVICORN_PORT"] = fmt.Sprintf("%d", port)
	}

	if micro.Presets != nil {
		for _, env := range micro.Presets.Env {
			// If the env is already set by the user, don't override it
			if os.Getenv(env.Name) != "" {
				continue
			}

			if env.Default == "" {
				continue
			}

			environ[env.Name] = env.Default
		}
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
	cmd.Stdout = writer.NewPrefixer(micro.Name, os.Stdout)
	cmd.Stderr = writer.NewPrefixer(micro.Name, os.Stderr)

	return cmd, nil
}

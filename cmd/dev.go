package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/internal/proxy"
	"github.com/deta/pc-cli/internal/runtime"
	"github.com/deta/pc-cli/internal/spacefile"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/shared"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	devProjectID  string
	devProjectDir string
	devCommand    string

	devCmd = &cobra.Command{
		Use:   "dev",
		Short: "run your app locally in dev mode",
		RunE:  dev,
	}
)

var (
	enginesToDevCommand = map[string]string{
		shared.React:     "npm run start",
		shared.Vue:       "npm run dev",
		shared.Svelte:    "npm run dev",
		shared.Next:      "npm run dev",
		shared.Nuxt:      "npm run dev",
		shared.SvelteKit: "npm run dev",
	}
)

func init() {
	devCmd.Flags().StringVarP(&devProjectID, "id", "i", "", "project id")
	devCmd.Flags().StringVarP(&devProjectDir, "dir", "d", "./", "root directory of the project")
	devCmd.Flags().StringVarP(&devCommand, "command", "c", "", "development command to run")
	rootCmd.AddCommand(devCmd)
}

func selectDevProjectID() (string, error) {
	promptInput := text.Input{
		Prompt:      "What is your Project ID?",
		Placeholder: "",
		Validator:   projectIDValidator,
	}

	return text.Run(&promptInput)
}

func CmdExec(command string, dir string, env []string, name string) {
	baseCommand, args, _ := strings.Cut(command, " ")

	cmd := exec.Command(baseCommand, strings.Split(args, " ")...)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}

	cmd.Dir = filepath.Join(cwd, dir)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)

	// redirect stdout and stderr to custom output
	cmd.Stdout = &customOutput{scope: name}
	cmd.Stderr = &customOutput{scope: name}

	cmd.Run()
}

type customOutput struct {
	scope string
}

// parse the logs and prefix them with the scope
func (c customOutput) Write(p []byte) (int, error) {
	normalized := strings.ReplaceAll(string(p), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	for _, line := range lines {
		fmt.Printf("[%s] %s\n", c.scope, line)
	}

	return len(p), nil
}

func formatEnvVars(key string, port string, name string, microType string, hostname string) []string {
	hostnameEnv := fmt.Sprintf("DETA_SPACE_APP_HOSTNAME=%s", hostname)
	microNameEnv := fmt.Sprintf("DETA_SPACE_APP_MICRO_NAME=%s", name)
	microTypeEnv := fmt.Sprintf("DETA_SPACE_APP_MICRO_TYPE=%s", microType)

	keyEnv := fmt.Sprintf("DETA_PROJECT_KEY=%s", key)
	portEnv := fmt.Sprintf("PORT=%s", port)

	envStrings := []string{hostnameEnv, microNameEnv, microTypeEnv, keyEnv, portEnv}

	return envStrings
}

// Normalize the path to a subtree path
func normalizePath(pathString string) string {
	p := path.Join("/", pathString, "/")
	if p == "/" {
		return p
	}

	return p + "/"
}

func generatePath(pathString *string, name string, primary bool) string {
	if pathString == nil {
		if primary {
			return "/"
		}

		return normalizePath(name)
	}

	return normalizePath(*pathString)
}

func getDevCommand(micro *shared.Micro) (string, error) {
	if micro.Dev != "" {
		return micro.Dev, nil
	}

	engine, ok := shared.EngineAliases[micro.Engine]
	if !ok {
		return "", fmt.Errorf("unsupported engine for %s", micro.Name)
	}

	command, ok := enginesToDevCommand[engine]
	if !ok {
		return "", fmt.Errorf("no dev command specified for %s", micro.Name)
	}

	return command, nil
}

func cleanup() {
	logger.Println("\nCleaning up...")
}

func dev(cmd *cobra.Command, args []string) error {
	logger.Println()

	e := make(chan os.Signal)
	signal.Notify(e, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-e
		cleanup()
		os.Exit(1)
	}()

	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	var err error

	// grab to port for the proxy server from the env or default to 8080
	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	hostname := fmt.Sprintf("localhost:%s", serverPort)

	devProjectDir = filepath.Clean(devProjectDir)

	runtimeManager, err := runtime.NewManager(&devProjectDir, false)
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
	} else if isFlagEmpty(devProjectID) {
		logger.Printf("No project was found in the current directory.\n\n")

		devProjectID, err = selectDevProjectID()
		if err != nil {
			return fmt.Errorf("problem while trying to get project id from prompt, %v", err)
		}
	}

	// check if spacefile is present
	isSpacefilePresent, err := spacefile.IsSpacefilePresent(pushProjectDir)
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
	s, err := spacefile.Open(projectDir)
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
	key, err := auth.GetProjectKey(devProjectID)
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

			key = r.Value

			// store the project key locally
			err = auth.StoreProjectKey(devProjectID, key)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		logger.Printf("%sUsing existing project key", emoji.Key)
	}

	// if the user has specified a dev command manually, just run it
	if *&devCommand != "" {
		logger.Printf("%sRunning development command...\n\n", emoji.Terminal)

		envVars := formatEnvVars(key, serverPort, "", "", hostname)

		defer CmdExec(*&devCommand, ".", envVars, "dev")
	} else {
		logger.Printf("%sStarting micro servers...\n\n", emoji.Terminal)

		mux := http.NewServeMux()

		port := 3000
		for _, micro := range s.Micros {
			logger.Printf("Micro \"%s\"", micro.Name)

			// Choose a port
			port += 1

			// Parse the micro's config
			command, err := getDevCommand(micro)
			if err != nil {
				return err
			}
			path := generatePath(micro.Path, micro.Name, micro.Primary)
			src := micro.Src
			var microType string
			if micro.Primary {
				microType = "primary"
			} else {
				microType = "normal"
			}

			// Format the env vars that will be passed to the micro
			envVars := formatEnvVars(key, fmt.Sprintf("%d", port), micro.Name, microType, hostname)

			logger.Printf("L port: %d\n", port)
			logger.Printf("L path: %s\n", path)
			logger.Printf("L command: %s\n\n", command)

			go CmdExec(command, src, envVars, micro.Name)

			// initialize a reverse proxy and pass the micros endpoint and path that will be proxied
			endpoint := fmt.Sprintf("http://localhost:%d", port)
			httpProxy, err := proxy.NewProxy(endpoint, path)
			if err != nil {
				return err
			}

			// handle all requests to the micros path using the proxy
			mux.HandleFunc(path, proxy.ProxyRequestHandler(httpProxy))
		}

		logger.Printf(styles.Green(fmt.Sprintf("Starting proxy server on port %s...\n", serverPort)))

		// start the proxy server
		err = http.ListenAndServe(":"+serverPort, mux)
		if err != nil {
			return err
		}
	}

	return nil
}

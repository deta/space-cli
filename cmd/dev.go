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
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	devProjectID  string
	devProjectDir string

	devCmd = &cobra.Command{
		Use:   "dev",
		Short: "run your app in dev mode",
		RunE:  dev,
	}
)

func init() {
	devCmd.Flags().StringVarP(&devProjectID, "id", "i", "", "project id of project dev")
	devCmd.Flags().StringVarP(&devProjectDir, "dir", "d", "./", "src of project to push")
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

func formatEnvVars(key string, port int) []string {
	type Envs struct {
		DETA_PROJECT_KEY string
		PORT             int
	}

	envs := Envs{key, port}

	keyEnvString := fmt.Sprintf("DETA_PROJECT_KEY=%s", envs.DETA_PROJECT_KEY)
	portEnvString := fmt.Sprintf("PORT=%d", envs.PORT)

	envStrings := []string{portEnvString, keyEnvString}

	return envStrings
}

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

func dev(cmd *cobra.Command, args []string) error {
	logger.Println()

	e := make(chan os.Signal)
	signal.Notify(e, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-e
		os.Exit(1)
	}()

	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	var err error

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
	isSpacefilePrsent, err := spacefile.IsSpacefilePresent(pushProjectDir)
	if err != nil {
		if errors.Is(err, spacefile.ErrSpacefileWrongCase) {
			logger.Printf("%s The Spacefile must be called exactly 'Spacefile'.\n", emoji.ErrorExclamation)
			return nil
		}
		return err
	}
	if !isSpacefilePrsent {
		logger.Println(styles.Errorf("%s No Spacefile is present. Please add a Spacefile.", emoji.ErrorExclamation))
		return nil
	}

	// parse spacefile and validate
	logger.Printf("Validating Spacefile...\n\n")

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

	logger.Printf("Checking for project key...\n")

	// check if we have already stored the project key based on the project's id
	key, err := auth.GetProjectKey(devProjectID)
	if err != nil {
		if errors.Is(err, auth.ErrNoProjectKeyFound) {
			logger.Printf("No project key found, generating new key...\n")

			hostname, err := os.Hostname()
			if err != nil {
				hostname = ""
			}

			// create a new project key using the api
			r, err := client.CreateProjectKey(&api.CreateProjectKeyRequest{
				AppID: devProjectID,
				Name:  "dev " + hostname,
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
	}

	logger.Printf("Using project key: %s\n\n", key)
	logger.Printf("Starting micro servers...\n\n")

	mux := http.NewServeMux()

	port := 3000
	for _, micro := range s.Micros {
		logger.Printf("Micro \"%s\"", micro.Name)

		// Choose a port
		port += 1

		// Grab the command and path from the micro's config
		command := micro.Dev
		path := generatePath(micro.Path, micro.Name, micro.Primary)
		src := micro.Src

		// Format the env vars that will be passed to the micro
		envVars := formatEnvVars(key, port)
		endpoint := fmt.Sprintf("http://localhost:%d", port)

		logger.Printf("L port: %d\n", port)
		logger.Printf("L path: %s\n", path)
		logger.Printf("L command: %s\n\n", command)

		go CmdExec(command, src, envVars, micro.Name)

		// initialize a reverse proxy and pass the micros endpoint and path that will be proxied
		httpProxy, err := proxy.NewProxy(endpoint, path)
		if err != nil {
			return err
		}

		// handle all requests to the micros path using the proxy
		mux.HandleFunc(path, proxy.ProxyRequestHandler(httpProxy))
	}

	// grab to port for the proxy from the env or default to 8080
	proxyPort := os.Getenv("PORT")
	if proxyPort == "" {
		proxyPort = "8080"
	}

	logger.Printf(styles.Green(fmt.Sprintf("Starting proxy server on port %s\n", proxyPort)))

	// start the proxy server
	err = http.ListenAndServe(":"+proxyPort, mux)
	if err != nil {
		return err
	}

	return nil
}

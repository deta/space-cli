package shared

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/deta/pc-cli/pkg/writer"
	"mvdan.cc/sh/v3/shell"
)

// supported engines
const (
	Static    = "static"
	React     = "react"
	Svelte    = "svelte"
	Vue       = "vue"
	Next      = "next"
	Nuxt      = "nuxt"
	SvelteKit = "svelte-kit"
	Python38  = "python3.8"
	Python39  = "python3.9"
	Node14x   = "nodejs14.x"
	Node16x   = "nodejs16.x"
	Custom    = "custom"
)

var (
	SupportedEngines = []string{Static, React, Svelte, Vue, Next, Nuxt, SvelteKit, Python38, Python39, Node14x, Node16x, Custom}

	EngineAliases = map[string]string{
		"static":     Static,
		"react":      React,
		"svelte":     Svelte,
		"vue":        Vue,
		"next":       Next,
		"nuxt":       Nuxt,
		"svelte-kit": SvelteKit,
		"python3.9":  Python39,
		"python3.8":  Python38,
		"nodejs14.x": Node14x,
		"nodejs14":   Node14x,
		"nodejs16.x": Node16x,
		"nodejs16":   Node16x,
		"custom":     Custom,
	}

	EnginesToRuntimes = map[string]string{
		Static:    Node14x,
		React:     Node14x,
		Svelte:    Node14x,
		Vue:       Node14x,
		Next:      Node16x,
		Nuxt:      Node16x,
		SvelteKit: Node16x,
		Python38:  Python38,
		Python39:  Python38,
		Node14x:   Node14x,
		Node16x:   Node16x,
		Custom:    Custom,
	}

	supportedFrontendEngines = map[string]struct{}{
		React:  {},
		Vue:    {},
		Svelte: {},
		Static: {},
	}

	supportedFullstackEngines = map[string]struct{}{
		Next:      {},
		Nuxt:      {},
		SvelteKit: {},
	}

	engineToDevCommand = map[string]string{
		React:     "npm run start -- --port $PORT",
		Vue:       "npm run dev -- --port $PORT",
		Svelte:    "npm run dev -- --port $PORT",
		Next:      "npm run dev -- --port $PORT",
		Nuxt:      "npm run dev -- --port $PORT",
		SvelteKit: "npm run dev -- --port $PORT",
	}
)

type ActionEvent struct {
	ID      string `json:"id"`
	Trigger string `json:"trigger"`
}

type ActionRequest struct {
	Event ActionEvent `json:"event"`
}

// Environment xx
type Environment struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
}

// Presets xx
type Presets struct {
	Env     []Environment `yaml:"env"`
	APIKeys bool          `yaml:"api_keys"`
}

// Action xx
type Action struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Trigger     string `yaml:"trigger"`
	Interval    string `yaml:"default_interval"`
	Path        string `yaml:"path"`
}

// Micro xx
type Micro struct {
	Name         string   `yaml:"name"`
	Src          string   `yaml:"src"`
	Engine       string   `yaml:"engine"`
	Path         string   `yaml:"path,omitempty"`
	Presets      *Presets `yaml:"presets,omitempty"`
	Public       bool     `yaml:"public,omitempty"`
	PublicRoutes []string `yaml:"public_routes,omitempty"`
	Primary      bool     `yaml:"primary"`
	Runtime      string   `yaml:"runtime,omitempty"`
	Commands     []string `yaml:"commands,omitempty"`
	Include      []string `yaml:"include,omitempty"`
	Actions      []Action `yaml:"actions,omitempty"`
	Serve        string   `yaml:"serve,omitempty"`
	Run          string   `yaml:"run,omitempty"`
	Dev          string   `yaml:"dev,omitempty"`
}

func (m Micro) Type() string {
	if m.Primary {
		return "primary"
	}
	return "normal"
}

var ErrNoDevCommand = errors.New("no dev command found for micro")

func (micro *Micro) Command(directory, projectKey string, port int) (*exec.Cmd, error) {
	var devCommand string

	if micro.Dev != "" {
		devCommand = micro.Dev
	} else if engineToDevCommand[micro.Engine] != "" {
		devCommand = engineToDevCommand[micro.Engine]
	} else {
		return nil, ErrNoDevCommand
	}

	commandDir := path.Join(directory, micro.Src)

	environ := map[string]string{
		"PORT":                      fmt.Sprintf("%d", port),
		"DETA_PROJECT_KEY":          projectKey,
		"DETA_SPACE_APP_HOSTNAME":   fmt.Sprintf("localhost:%d", port),
		"DETA_SPACE_APP_MICRO_NAME": micro.Name,
		"DETA_SPACE_APP_MICRO_TYPE": micro.Type(),
	}

	if micro.Presets != nil {
		for _, env := range micro.Presets.Env {
			// If the env is already set by the user, don't override it
			if os.Getenv(env.Name) != "" {
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

func IsFrontendEngine(engine string) bool {
	_, ok := supportedFrontendEngines[engine]
	return ok
}

func IsFullstackEngine(engine string) bool {
	_, ok := supportedFullstackEngines[engine]
	return ok
}

type DiscoveryFrontmatter struct {
	Title      string `yaml:"title,omitempty" json:"title"`
	Tagline    string `yaml:"tagline,omitempty" json:"tagline"`
	ThemeColor string `yaml:"theme_color,omitempty" json:"theme_color"`
	Git        string `yaml:"git,omitempty" json:"git"`
	Homepage   string `yaml:"homepage,omitempty" json:"homepage"`
	ContentRaw string `yaml:"content_raw,omitempty" json:"content_raw"`
}

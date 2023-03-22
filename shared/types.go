package shared

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
)

// Environment xx
type Environment struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
}

// Presets xx
type Presets struct {
	Env     []*Environment `yaml:"env"`
	APIKeys bool           `yaml:"api_keys"`
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
	Name         string    `yaml:"name"`
	Src          string    `yaml:"src"`
	Engine       string    `yaml:"engine"`
	Path         *string   `yaml:"path,omitempty"`
	Presets      *Presets  `yaml:"presets,omitempty"`
	Public       bool      `yaml:"public,omitempty"`
	PublicRoutes []string  `yaml:"public_routes,omitempty"`
	Primary      bool      `yaml:"primary"`
	Runtime      string    `yaml:"runtime,omitempty"`
	Commands     []string  `yaml:"commands,omitempty"`
	Include      []string  `yaml:"include,omitempty"`
	Actions      []*Action `yaml:"actions,omitempty"`
	Serve        string    `yaml:"serve,omitempty"`
	Run          string    `yaml:"run,omitempty"`
}

func IsFrontendEngine(engine string) bool {
	_, ok := supportedFrontendEngines[engine]
	return ok
}

func IsFullstackEngine(engine string) bool {
	_, ok := supportedFullstackEngines[engine]
	return ok
}

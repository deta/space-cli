package shared

// Environment xx
type Environment struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
}

// Presets xx
type Presets struct {
	Env []*Environment `yaml:"env"`
}

// Micro xx
type Micro struct {
	Name         string              `yaml:"name"`
	Src          string              `yaml:"src"`
	Engine       string              `yaml:"engine"`
	Path         *string             `yaml:"path,omitempty"`
	Presets      *Presets            `yaml:"presets,omitempty"`
	PublicRoutes map[string][]string `yaml:"public_routes,omitempty"`
	Primary      bool                `yaml:"primary"`
	Runtime      string              `yaml:"runtime,omitempty"`
	Commands     []string            `yaml:"commands,omitempty"`
	AppRoot      string              `yaml:"approot,omitempty"`
	Artefact     string              `yaml:"artefact,omitempty"`
	Run          string              `yaml:"run,omitempty"`
}

// supported engines
const (
	Static    = "static"
	React     = "create-react-app"
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
	Go        = "go"
	Rust      = "rust"
)

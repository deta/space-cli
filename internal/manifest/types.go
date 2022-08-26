package manifest

import (
	"errors"
)

var (
	// ErrManifestNotFound manifest file not found
	ErrManifestNotFound = errors.New("manifest file not found")
)

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

// Manifest xx
type Manifest struct {
	V      int      `yaml:"v"`
	Icon   string   `yaml:"icon,omitempty"`
	Micros []*Micro `yaml:"micros"`
}

func getSupportedManifestNames() []string {
	return []string{"deta.yml", "deta.yaml", "deta_manifest.yml", "deta_manifest.yaml"}
}

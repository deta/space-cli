package manifest

import (
	"errors"

	"github.com/deta/pc-cli/shared"
)

var (
	// ErrManifestNotFound manifest file not found
	ErrManifestNotFound = errors.New("manifest file not found")
)

// Manifest xx
type Manifest struct {
	V      int             `yaml:"v"`
	Icon   string          `yaml:"icon,omitempty"`
	Micros []*shared.Micro `yaml:"micros,omitempty"`
}

func getSupportedManifestNames() []string {
	return []string{"space.yml", "space.yaml"}
}

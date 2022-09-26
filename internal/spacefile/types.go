package spacefile

import (
	"errors"

	"github.com/deta/pc-cli/shared"
)

var (
	// ErrSpacefileNotFound spacefile file not found
	ErrSpacefileNotFound = errors.New("spacefile file not found")
)

// Spacefile xx
type Spacefile struct {
	V      int             `yaml:"v"`
	Icon   string          `yaml:"icon,omitempty"`
	Micros []*shared.Micro `yaml:"micros,omitempty"`
}

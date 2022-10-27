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
	V       int             `yaml:"v"`
	Icon    string          `yaml:"icon,omitempty"`
	AppName string          `yaml:"app_name,omitempty"`
	Micros  []*shared.Micro `yaml:"micros,omitempty"`
}

// IconMeta xx
type IconMeta struct {
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// Icon xx
type Icon struct {
	Raw      []byte    `json:"icon"`
	IconMeta *IconMeta `json:"icon_meta"`
}

// MicroError xx
type MicroError struct {
	Err   error
	Micro *shared.Micro
}

func (me *MicroError) Error() string {
	return me.Err.Error()
}

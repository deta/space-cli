package scanner

import (
	"errors"
	"os"

	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/shared"
)

var (
	// ErrEmptyMicroName empty micro name
	ErrEmptyMicroName = errors.New("empty micro name")

	// ErrEmptyMicroSrc empty micro src
	ErrEmptyMicroSrc = errors.New("empty micro src")

	// ErrEmptyMicroEngine empty micro engine
	ErrEmptyMicroEngine = errors.New("empty micro engine")

	// ErrInvalidMicroSrc cannot find folder for micro
	ErrInvalidMicroSrc = errors.New("invalid micro src")

	// ErrInvalidIcon cannot find icon path
	ErrInvalidIcon = errors.New("invalid icon path")
)

func ValidateManifestIcon(manifest *manifest.Manifest) error {
	if manifest.Icon == "" {
		return nil
	}

	_, err := os.Stat(manifest.Icon)
	if os.IsNotExist(err) {
		return ErrInvalidIcon
	}

	return nil
}

func ValidateMicro(micro *shared.Micro) error {
	if micro.Name == "" {
		return ErrEmptyMicroName
	}

	if micro.Src == "" {
		return ErrEmptyMicroSrc
	}

	if micro.Engine == "" {
		return ErrEmptyMicroEngine
	}

	_, err := os.Stat(micro.Src)
	if os.IsNotExist(err) {
		return ErrInvalidMicroSrc
	}
	return nil
}

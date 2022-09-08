package scanner

import (
	"errors"
	"os"

	"github.com/deta/pc-cli/internal/manifest"
	"github.com/deta/pc-cli/shared"
	"golang.org/x/exp/slices"
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

	// ErrInvalidMicroEngine
	ErrInvalidMicroEngine = errors.New("invalid micro engine")

	// ErrInvalidIcon cannot find icon path
	ErrInvalidIcon = errors.New("invalid icon path")

	// ErrDuplicateMicros
	ErrDuplicateMicros = errors.New("micro names have to be unique")

	// ErrExceedsMaxMicroLimit
	ErrExceedsMaxMicroLimit = errors.New("manifest exceeds max micro limit of 5 micros")

	// ErrNoPrimaryMicro
	ErrNoPrimaryMicro = errors.New("no primary micro present")
)

type MicroError struct {
	Err   error
	Micro *shared.Micro
}

func (me *MicroError) Error() string {
	return me.Err.Error()
}

// ValidateManifest checks for general errors such as duplicate micros and max micro limit
func ValidateManifest(m *manifest.Manifest) []error {

	var isPrimaryMicroPresent bool

	// microNames used to check if micros are unique
	microNames := make(map[string]struct{})

	errors := []error{}

	if len(m.Micros) > 5 {
		errors = append(errors, ErrExceedsMaxMicroLimit)
	}

	for _, micro := range m.Micros {
		if _, ok := microNames[micro.Name]; ok {
			errors = append(errors, ErrDuplicateMicros)
		}
		if micro.Primary {
			isPrimaryMicroPresent = true
		}
		microNames[micro.Name] = struct{}{}
		microErrors := ValidateMicro(micro)
		for _, err := range microErrors {
			if err != nil {
				errors = append(errors, &MicroError{Err: err, Micro: micro})
			}
		}
	}

	if !isPrimaryMicroPresent && len(m.Micros) > 1 {
		errors = append(errors, ErrNoPrimaryMicro)
	}

	return errors
}

func ValidateManifestIcon(icon string) error {
	if icon == "" {
		return nil
	}

	_, err := os.Stat(icon)
	if os.IsNotExist(err) {
		return ErrInvalidIcon
	}

	return nil
}

func ValidateMicro(micro *shared.Micro) []error {
	errors := []error{}

	if micro.Name == "" {
		errors = append(errors, ErrEmptyMicroName)
	}

	if !slices.Contains(shared.SupportedEngines, micro.Engine) {
		if micro.Engine == "" {
			errors = append(errors, ErrEmptyMicroEngine)
		} else {
			errors = append(errors, ErrInvalidMicroEngine)
		}
	}

	_, err := os.Stat(micro.Src)
	if os.IsNotExist(err) {
		if micro.Src == "" {
			errors = append(errors, ErrEmptyMicroSrc)
		} else {
			errors = append(errors, ErrInvalidMicroSrc)
		}
	}

	return errors
}

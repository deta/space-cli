package spacefile

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"path"

	"os"

	_ "golang.org/x/image/webp"

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

	// ErrInvalidMicroEngine
	ErrInvalidMicroEngine = errors.New("invalid micro engine")

	// ErrInvalidIconPath cannot find icon path
	ErrInvalidIconPath = errors.New("invalid icon path")

	// ErrInvalidIconType
	ErrInvalidIconType = errors.New("invalid icon type")

	// ErrInvalidIconSize
	ErrInvalidIconSize = errors.New("invalid icon size")

	// ErrDuplicateMicros
	ErrDuplicateMicros = errors.New("micro names have to be unique")

	// ErrExceedsMaxMicroLimit
	ErrExceedsMaxMicroLimit = errors.New("spacefile exceeds max micro limit of 5 micros")

	// ErrNoPrimaryMicro
	ErrNoPrimaryMicro = errors.New("no primary micro present")

	// ErrAppNameMaxLengthExceeded
	ErrAppNameMaxLengthExceeded = errors.New("app_name is too long, max length is 16 characters")

	// MaxIconWidth
	MaxIconWidth = 512

	// MaxIconHeight
	MaxIconHeight = 512

	// MaxIconSize
	MaxIconSize = MaxIconHeight * MaxIconWidth
)

// ValidateSpacefile checks for general errors such as duplicate micros and max micro limit
func ValidateSpacefile(s *Spacefile, projectDir string) []error {

	var primarySpecified bool

	// microNames used to check if micros are unique
	microNames := make(map[string]struct{})

	errors := []error{}

	err := ValidateSpacefileIcon(s.Icon)
	if err != nil {
		errors = append(errors, err)
	}

	if len(s.Micros) > 5 {
		errors = append(errors, ErrExceedsMaxMicroLimit)
	}

	if len(s.AppName) > 16 {
		errors = append(errors, ErrAppNameMaxLengthExceeded)
	}
	for _, micro := range s.Micros {
		if _, ok := microNames[micro.Name]; ok {
			errors = append(errors, ErrDuplicateMicros)
		}
		if micro.Primary {
			primarySpecified = true
		}
		microNames[micro.Name] = struct{}{}
		microErrors := ValidateMicro(micro, projectDir)
		for _, err := range microErrors {
			if err != nil {
				errors = append(errors, &MicroError{Err: err, Micro: micro})
			}
		}
	}

	if !primarySpecified && len(s.Micros) > 1 {
		errors = append(errors, ErrNoPrimaryMicro)
	}

	return errors
}

// ValidateSpacefileIcon validate spacefile icon
func ValidateSpacefileIcon(iconPath string) error {
	if iconPath == "" {
		return nil
	}

	_, err := os.Stat(iconPath)
	if os.IsNotExist(err) {
		return ErrInvalidIconPath
	}

	iconMeta, err := getIconMeta(iconPath)
	if err != nil {
		return err
	}

	if iconMeta.ContentType != "image/png" && iconMeta.ContentType != "image/webp" {
		return ErrInvalidIconType
	}

	if iconMeta.Height != MaxIconHeight && iconMeta.Width != MaxIconWidth {
		return ErrInvalidIconSize
	}

	return nil
}

// ValidateMicro validate micro
func ValidateMicro(micro *shared.Micro, projectDir string) []error {
	errors := []error{}

	if micro.Name == "" {
		errors = append(errors, ErrEmptyMicroName)
	}

	if _, ok := shared.EngineAliases[micro.Engine]; !ok {
		if micro.Engine == "" {
			errors = append(errors, ErrEmptyMicroEngine)
		} else {
			errors = append(errors, ErrInvalidMicroEngine)
		}
	}

	if micro.Src == "" {
		errors = append(errors, ErrEmptyMicroSrc)
	} else {
		if _, err := os.Stat(path.Join(projectDir, micro.Src)); os.IsNotExist(err) {
			errors = append(errors, ErrInvalidMicroSrc)
		}
	}

	if micro.Serve != "" && !shared.IsFrontendEngine(micro.Engine) {
		errors = append(errors, fmt.Errorf("serve is only valid for frontend engines"))
	}

	if micro.Serve != "" && len(micro.Include) > 0 {
		errors = append(errors, fmt.Errorf("cannot use both serve and include"))
	}

	if len(micro.Include) > 0 && micro.Serve != "" {
		errors = append(errors, fmt.Errorf("cannot use both serve and include"))
	}

	if shared.IsFrontendEngine(micro.Engine) && len(micro.Include) > 0 {
		errors = append(errors, fmt.Errorf("include is not valid for frontend engines"))
	}

	return errors
}

func getIconMeta(iconPath string) (*IconMeta, error) {

	_, err := os.Stat(iconPath)
	if os.IsNotExist(err) {
		return nil, ErrInvalidIconPath
	}

	imgFile, err := os.Open(iconPath)
	if err != nil {
		return nil, ErrInvalidIconPath
	}
	defer imgFile.Close()

	imgMeta, imgType, err := image.DecodeConfig(imgFile)
	if err != nil {
		if errors.Is(image.ErrFormat, err) {
			return nil, ErrInvalidIconType
		}
		return nil, ErrInvalidIconPath
	}

	return &IconMeta{
		Width:       imgMeta.Width,
		Height:      imgMeta.Height,
		ContentType: mime.TypeByExtension("." + imgType),
	}, nil
}

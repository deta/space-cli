package spacefile

import (
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"os"
	"path/filepath"
)

var (
	// ErrInvalidIconPath cannot find icon path
	ErrInvalidIconPath = errors.New("invalid icon path")

	// ErrInvalidIconType
	ErrInvalidIconType = errors.New("invalid icon type")

	// ErrInvalidIconSize
	ErrInvalidIconSize = errors.New("invalid icon size")

	// MaxIconWidth
	MaxIconWidth = 512

	// MaxIconHeight
	MaxIconHeight = 512

	// MaxIconSize
	MaxIconSize = MaxIconHeight * MaxIconWidth
)

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

// ValidateSpacefileIcon validate spacefile icon
func ValidateIcon(iconPath string) error {
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

func getIconMeta(iconPath string) (*IconMeta, error) {
	abs, _ := filepath.Abs(iconPath)
	imgFile, err := os.Open(abs)
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

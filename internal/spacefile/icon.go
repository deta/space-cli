package spacefile

import (
	"errors"
	"image"
	"mime"
	"os"
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

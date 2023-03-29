package spacefile

import (
	"errors"
	"testing"
)

func TestIconDoesNotExists(t *testing.T) {
	iconPath := "./testdata/icons/does-not-exists.png"
	err := ValidateIcon(iconPath)
	if !errors.Is(err, ErrInvalidIconPath) {
		t.Fatalf("expected error %v but got %v", ErrInvalidIconPath, err)
	}
}

func TestIconInvalidSize(t *testing.T) {
	iconPath := "./testdata/icons/size-128.png"
	err := ValidateIcon(iconPath)
	if !errors.Is(err, ErrInvalidIconSize) {
		t.Fatalf("expected error %v but got %v", ErrInvalidIconPath, err)
	}
}

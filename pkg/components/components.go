package components

import "errors"

var (
	// ErrPromptCancelled is returned when prompt is cancelled
	ErrPromptCancelled = errors.New("prompt cancelled")
)

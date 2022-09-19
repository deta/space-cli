package emoji

import (
	"os"
	"runtime"
	"syscall"

	"golang.org/x/term"
)

type Emoji struct {
	Emoji    string
	Fallback string
}

func (e Emoji) String() string {

	if SupportsEmoji() {
		return e.Emoji
	}

	return e.Fallback
}

func SupportsEmoji() bool {

	if !term.IsTerminal(int(syscall.Stdout)) {
		return false
	}

	platform := runtime.GOOS
	switch platform {
	case "windows":
		_, isWindowsTerminal := os.LookupEnv("WT_SESSION")
		return isWindowsTerminal
	case "darwin":
		return true
	case "linux":
		return false
	}

	return false
}

package runtime

import (
	"strings"
	"testing"

	ignore "github.com/sabhiram/go-gitignore"
)

var files = map[string]bool{
	".git":                  true,
	".env":                  true,
	".envrc":                true,
	"venv":                  true,
	"virtualenv":            true,
	"venv/main.py":          true,
	"node_modules/index.js": true,
	"folder/.env":           true,
	"main.py":               false,
}

func TestIgnorePatterns(t *testing.T) {
	lines := strings.Split(defaultSpaceignore, "\n")

	spaceignore := ignore.CompileIgnoreLines(lines...)

	for file, shouldBeIgnored := range files {
		if spaceignore.MatchesPath(file) != shouldBeIgnored {
			t.Fatalf("expected %s to be ignored: %t", file, shouldBeIgnored)
		}

	}
}

func TestAddNewPattern(t *testing.T) {
	lines := strings.Split(defaultSpaceignore, "\n")
	lines = append(lines, "main.py")
	spaceignore := ignore.CompileIgnoreLines(lines...)

	if !spaceignore.MatchesPath("main.py") {
		t.Fatalf("expected main.py to not be ignored")
	}
}

func TestOverrideExistingPattern(t *testing.T) {
	lines := strings.Split(defaultSpaceignore, "\n")
	lines = append(lines, "!.env")
	spaceignore := ignore.CompileIgnoreLines(lines...)

	if spaceignore.MatchesPath(".env") {
		t.Fatalf("expected venv/main.py to not be ignored")
	}
}

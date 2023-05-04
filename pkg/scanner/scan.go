package scanner

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/deta/space/shared"
)

func Scan(sourceDir string) ([]*shared.Micro, error) {
	var micros []*shared.Micro

	// scan root source dir for a micro
	m, err := scanDir(sourceDir)
	if err != nil {
		return nil, err
	}
	if m != nil {
		// root folder has a micro return as a single micro app
		micros = append(micros, m)
		return micros, nil
	}

	// scan subfolders for micros
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			m, err = scanDir(filepath.Join(sourceDir, entry.Name()))
			if err != nil {
				return nil, err
			}
			if m != nil {
				micros = append(micros, m)
			}
		}
	}

	return micros, nil
}

func scanDir(dir string) (*shared.Micro, error) {
	runtimeScanners := []engineScanner{
		pythonScanner,
		nodeScanner,
		goScanner,
		rustScanner,
		staticScanner,
	}

	for _, scanner := range runtimeScanners {
		m, err := scanner(dir)
		if err != nil {
			return nil, err
		}
		if m != nil {
			return m, nil
		}
	}
	return nil, nil
}

var nonAlphaNumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func cleanMicroName(name string) string {
	return nonAlphaNumeric.ReplaceAllString(name, "-")
}

func getMicroNameFromPath(dir string) (string, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	base := filepath.Base(absPath)
	return cleanMicroName(base), nil
}

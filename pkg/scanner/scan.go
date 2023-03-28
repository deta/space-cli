package scanner

import (
	"os"
	"path/filepath"

	"github.com/deta/pc-cli/types"
)

func Scan(sourceDir string) ([]*types.Micro, error) {
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, err
	}

	var micros []*types.Micro

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
	for _, file := range files {
		if file.IsDir() {
			m, err = scanDir(filepath.Join(sourceDir, file.Name()))
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

func scanDir(dir string) (*types.Micro, error) {
	runtimeScanners := []engineScanner{
		pythonScanner,
		nodeScanner,
		goScanner,
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

func getMicroNameFromPath(dir string) (string, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	return filepath.Base(absPath), nil
}

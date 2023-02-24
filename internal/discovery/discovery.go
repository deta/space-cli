package discovery

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/deta/pc-cli/pkg/util/fs"
	"github.com/deta/pc-cli/shared"
	"gopkg.in/yaml.v2"
)

const (
	// DiscoveryFilename discovery filename
	DiscoveryFilename = "Discovery.md"
)

var (
	// ErrDiscoveryFileNotFound dicovery file not found
	ErrDiscoveryFileNotFound = errors.New("discovery file not found")
	// ErrDiscoveryFileWrongCase discovery file wrong case
	ErrDiscoveryFileWrongCase = errors.New("discovery file wrong case")
)

func checkDiscoveryFileCase(sourceDir string) (string, bool, error) {
	files, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		return "", false, err
	}
	for _, f := range files {
		if strings.ToLower(f.Name()) == strings.ToLower(DiscoveryFilename) {
			if f.Name() != DiscoveryFilename {
				return f.Name(), false, nil
			}
			return f.Name(), true, nil
		}
	}
	return "", false, ErrDiscoveryFileNotFound
}

// Open open discovery file
func Open(sourceDir string) ([]byte, error) {
	var exists bool
	var err error

	exists, err = fs.FileExists(sourceDir, DiscoveryFilename)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrDiscoveryFileNotFound
	}

	existingDiscoveryFileName, correctCase, err := checkDiscoveryFileCase(sourceDir)
	if err != nil {
		return nil, err
	}

	if !correctCase {
		return nil, fmt.Errorf("'%s' must be called exactly %s", existingDiscoveryFileName, DiscoveryFilename)
	}

	// read raw contents from discovery file
	c, err := ioutil.ReadFile(filepath.Join(sourceDir, DiscoveryFilename))
	if err != nil {
		return nil, fmt.Errorf("failed to read contents of discovery file: %w", err)
	}

	return c, nil
}

func CreateDiscoveryFile(filename string, discovery shared.DiscoveryFrontmatter) error {
	f, err := os.Create(filename)
	if err != nil {
		f.Close()
		return err
	}

	js, _ := yaml.Marshal(discovery)
	fmt.Fprintln(f, "---")
	fmt.Fprint(f, string(js))
	fmt.Fprintln(f, "---")

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

package discovery

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/deta/pc-cli/pkg/util/fs"
)

const (
	// DiscoveryFilename discovery filename
	DiscoveryFilename = "discovery.md"
)

var (
	// ErrDiscoveryFileNotFound dicovery file not found
	ErrDiscoveryFileNotFound = errors.New("discovery file not found")
)

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

	// read raw contents from discovery file
	c, err := ioutil.ReadFile(filepath.Join(sourceDir, DiscoveryFilename))
	if err != nil {
		return nil, fmt.Errorf("failed to read contents of discovery file: %w", err)
	}

	return c, nil
}

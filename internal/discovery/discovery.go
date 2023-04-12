package discovery

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/deta/space/shared"
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
	files, err := os.ReadDir(sourceDir)
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

func CreateDiscoveryFile(name string, discovery shared.DiscoveryData) error {
	f, err := os.Create(name)
	if err != nil {
		f.Close()
		return err
	}

	js, _ := yaml.Marshal(discovery)
	fmt.Fprintln(f, "---")
	fmt.Fprint(f, string(js))
	fmt.Fprintln(f, "---")
	fmt.Fprintln(f)
	fmt.Fprintln(f, discovery.ContentRaw)

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

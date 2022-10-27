package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	// drwxrw----
	dirPermMode = 0760
	// -rw-rw---
	filePermMode = 0660
)

var (
	spaceDir        = ".space"
	projectMetaFile = "meta"
)

// Manager runtime manager handles files management and other services
type Manager struct {
	rootDir         string // working directory of the project
	spacePath       string // dir for storing project meta
	projectMetaPath string // path to info file about the project
}

// NewManager returns a new manager for the root dir of the project
// if initDirs is true, it creates dirs under root
func NewManager(root *string, initDirs bool) (*Manager, error) {
	var rootDir string
	if root != nil {
		rootDir = *root
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		rootDir = wd
	}

	spacePath := filepath.Join(rootDir, spaceDir)
	if initDirs {
		err := os.MkdirAll(spacePath, dirPermMode)
		if err != nil {
			return nil, err
		}
	}

	manager := &Manager{
		rootDir:         rootDir,
		spacePath:       spacePath,
		projectMetaPath: filepath.Join(spacePath, projectMetaFile),
	}

	return manager, nil
}

// reads the contents of a file, returns contents
func (m *Manager) readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

// StoreProjectMeta stores project meta to disk
func (m *Manager) StoreProjectMeta(p *ProjectMeta) error {
	marshalled, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(m.projectMetaPath, marshalled, filePermMode)
}

// GetProjectMeta gets the project info stored
func (m *Manager) GetProjectMeta() (*ProjectMeta, error) {
	contents, err := m.readFile(m.projectMetaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	projectMeta, err := projectMetaFromBytes(contents)
	if err != nil {
		return nil, err
	}

	return projectMeta, nil
}

func (m *Manager) IsProjectInitialized() (bool, error) {
	_, err := os.Stat(m.projectMetaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// AddSpaceToGitignore add .space to .gitignore
func (m *Manager) AddSpaceToGitignore() error {
	gitignorePath := filepath.Join(m.rootDir, ".gitignore")
	_, err := os.Stat(gitignorePath)
	gitignoreExists := true
	if err != nil {
		if os.IsNotExist(err) {
			gitignoreExists = false
		}
	}

	if gitignoreExists {
		contents, err := m.readFile(gitignorePath)
		if err != nil {
			return fmt.Errorf("failed to append .space to .gitignore: %w", err)
		}
		contents = append(contents, []byte("\n.space")...)
		err = ioutil.WriteFile(gitignorePath, contents, filePermMode)
		if err != nil {
			return fmt.Errorf("failed to append .space to .gitignore: %w", err)
		}
		return nil
	}

	err = ioutil.WriteFile(gitignorePath, []byte(".space"), filePermMode)
	if err != nil {
		return fmt.Errorf("failed to write .space to .gitignore: %w", err)
	}
	return nil
}

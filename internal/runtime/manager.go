package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	ignore "github.com/deta/pc-cli/pkg/ignore"
)

const (
	// drwxrw----
	dirPermMode = 0760
	// -rw-rw---
	filePermMode = 0660
)

var (
	PythonSkipPattern = `__pycache__`

	NodeSkipPattern = `node_modules`

	HiddenFilePattern = `.*`

	ignoreFile      = ".spaceignore"
	spaceDir        = ".space"
	projectMetaFile = "meta"

	defaultSkipPatterns = []string{HiddenFilePattern, PythonSkipPattern, NodeSkipPattern}
)

// Manager runtime manager handles files management and other services
type Manager struct {
	rootDir         string            // working directory of the project
	spacePath       string            // dir for storing project meta
	projectMetaPath string            // path to info file about the project
	skipPaths       *ignore.GitIgnore // skip patterns that need to be ignored for space push
	ignorePath      string            // .spaceignore path
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
		ignorePath:      filepath.Join(rootDir, ignoreFile),
		skipPaths:       nil,
	}

	// not handling error as we don't want cli to crash if .spaceignore is not found
	manager.handleIgnoreFile()

	return manager, nil
}

// handleIgnoreFile build a slice of patterns to skip from .spaceignore file if it exists
func (m *Manager) handleIgnoreFile() error {
	contents, err := m.readFile(m.ignorePath)
	if err != nil {
		return err
	}

	lines, err := readLines(contents)
	if err != nil {
		return err
	}

	m.skipPaths = ignore.CompileIgnoreLines(lines...)

	return nil
}

// StoreProjectMeta stores project meta to disk
func (m *Manager) StoreProjectMeta(p *ProjectMeta) error {
	marshalled, err := json.Marshal(p)
	if err != nil {
		return err
	}

	spaceReadmeNotes := "Don't commit this folder (.space) to git as it may contain security-sensitive data."
	ioutil.WriteFile(filepath.Join(m.spacePath, "README"), []byte(spaceReadmeNotes), filePermMode)

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
	gitignoreExists := true

	_, err := os.Stat(gitignorePath)
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

		// check if .space already exists
		pass, _ := regexp.MatchString(`(?m)^(\.space)\b`, string(contents))
		if pass {
			return nil
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

func CreateIgnoreFile(filename string, lines []string) error {
	f, err := os.Create(filename)
	if err != nil {
		f.Close()
		return err
	}

	for _, v := range lines {
		fmt.Fprintln(f, v)
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) CreateDefaultSpaceIgnoreFile() error {
	return CreateIgnoreFile(ignoreFile, defaultSkipPatterns)
}

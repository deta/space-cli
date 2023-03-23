package runtime

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

const (
	// drwxrw----
	dirPermMode = 0760
	// -rw-rw---
	filePermMode = 0660
)

var (
	PythonSkipPattern = `__pycache__`
	NodeSkipPattern   = `node_modules`
	ignoreFile        = ".spaceignore"
	spaceDir          = ".space"
	projectMetaFile   = "meta"
)

//go:embed .spaceignore
var defaultSpaceignore []byte

// Manager runtime manager handles files management and other services
type Manager struct {
	rootDir         string // working directory of the project
	spacePath       string // dir for storing project meta
	projectMetaPath string // path to info file about the project
}

// NewManager returns a new manager for the root dir of the project
// if initDirs is true, it creates dirs under root
func NewManager(root string) (*Manager, error) {
	var rootDir string
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rootDir = wd

	spacePath := filepath.Join(rootDir, spaceDir)
	manager := &Manager{
		rootDir:         rootDir,
		spacePath:       spacePath,
		projectMetaPath: filepath.Join(spacePath, projectMetaFile),
	}

	return manager, nil
}

func CreateSpaceignore(projectDir string) error {
	return os.WriteFile(path.Join(projectDir, ignoreFile), defaultSpaceignore, filePermMode)
}

// StoreProjectMeta stores project meta to disk
func StoreProjectMeta(projectDir string, p *ProjectMeta) error {
	spaceDir := path.Join(projectDir, ".space")
	if _, err := os.Stat(spaceDir); os.IsNotExist(err) {
		os.MkdirAll(spaceDir, dirPermMode)
	}
	marshalled, err := json.Marshal(p)
	if err != nil {
		return err
	}

	spaceReadmeNotes := "Don't commit this folder (.space) to git as it may contain security-sensitive data."
	os.WriteFile(filepath.Join(spaceDir, "README"), []byte(spaceReadmeNotes), filePermMode)

	return os.WriteFile(spaceDir, marshalled, filePermMode)
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
func AddSpaceToGitignore(projectDir string) error {
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		err = os.WriteFile(gitignorePath, []byte(".space"), filePermMode)
		if err != nil {
			return fmt.Errorf("failed to write .space to .gitignore: %w", err)
		}
		return nil
	}

	contents, err := os.ReadFile(gitignorePath)
	if err != nil {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	// check if .space already exists
	pass, _ := regexp.MatchString(`(?m)^(\.space)\b`, string(contents))
	if pass {
		return nil
	}

	contents = append(contents, []byte("\n.space")...)
	err = os.WriteFile(gitignorePath, contents, filePermMode)
	if err != nil {
		return fmt.Errorf("failed to append .space to .gitignore: %w", err)
	}
	return nil

}

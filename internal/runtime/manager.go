package runtime

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
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
var defaultSpaceignoreRaw string
var defaultSpaceIgnore *ignore.GitIgnore

func init() {
	lines := strings.Split(defaultSpaceignoreRaw, "\n")
	defaultSpaceIgnore = ignore.CompileIgnoreLines(lines...)
}

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

func CreateSpaceIgnoreIfNotExists(projectDir string) error {
	spaceIgnorePath := filepath.Join(projectDir, ignoreFile)
	if _, err := os.Stat(spaceIgnorePath); !os.IsNotExist(err) {
		return nil
	}

	return os.WriteFile(spaceIgnorePath, []byte(defaultSpaceignoreRaw), filePermMode)
}

// StoreProjectMeta stores project meta to disk
func (m *Manager) StoreProjectMeta(p *ProjectMeta) error {
	marshalled, err := json.Marshal(p)
	if err != nil {
		return err
	}

	spaceReadmeNotes := "Don't commit this folder (.space) to git as it may contain security-sensitive data."
	os.WriteFile(filepath.Join(m.spacePath, "README"), []byte(spaceReadmeNotes), filePermMode)

	return os.WriteFile(m.projectMetaPath, marshalled, filePermMode)
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
		err = os.WriteFile(gitignorePath, contents, filePermMode)
		if err != nil {
			return fmt.Errorf("failed to append .space to .gitignore: %w", err)
		}
		return nil
	}

	err = os.WriteFile(gitignorePath, []byte(".space"), filePermMode)
	if err != nil {
		return fmt.Errorf("failed to write .space to .gitignore: %w", err)
	}
	return nil
}

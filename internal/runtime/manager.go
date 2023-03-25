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
var defaultSpaceignore string

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

	return os.WriteFile(path.Join(spaceDir, projectMetaFile), marshalled, filePermMode)
}

// GetProjectMeta gets the project info stored
func GetProjectMeta(projectDir string) (*ProjectMeta, error) {
	contents, err := os.ReadFile(path.Join(projectDir, spaceDir, projectMetaFile))
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

func IsProjectInitialized(projectDir string) bool {
	_, err := os.Stat(path.Join(projectDir, spaceDir))
	return err == nil
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

	contents = append(contents, []byte("\n# Deta Space\n.space")...)
	err = os.WriteFile(gitignorePath, contents, filePermMode)
	if err != nil {
		return fmt.Errorf("failed to append .space to .gitignore: %w", err)
	}
	return nil

}

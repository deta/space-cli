package runtime

import (
	"encoding/json"
	"errors"
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
	PythonSkipPattern = `__pycache__`

	NodeSkipPattern = `node_modules`

	spaceDir        = ".space"
	projectMetaFile = "meta"
)

// StoreProjectMeta stores project meta to disk
func StoreProjectMeta(projectDir string, p *ProjectMeta) error {
	if _, err := os.Stat(filepath.Join(projectDir, spaceDir)); os.IsNotExist(err) {
		err = os.Mkdir(filepath.Join(projectDir, spaceDir), dirPermMode)
		if err != nil {
			return err
		}
	}
	marshalled, err := json.Marshal(p)
	if err != nil {
		return err
	}

	spaceReadmeNotes := "Don't commit this folder (.space) to git as it may contain security-sensitive data."
	ioutil.WriteFile(filepath.Join(projectDir, spaceDir, "README"), []byte(spaceReadmeNotes), filePermMode)

	// Add a .gitignore file so .space is not committed to git
	ioutil.WriteFile(filepath.Join(projectDir, spaceDir, ".gitignore"), []byte("*\n"), filePermMode)

	return ioutil.WriteFile(filepath.Join(projectDir, spaceDir, projectMetaFile), marshalled, filePermMode)
}

func GetProjectID(projectDir string) (string, error) {
	projectMeta, err := GetProjectMeta(projectDir)
	if err != nil {
		return "", err
	}
	return projectMeta.ID, nil
}

// GetProjectMeta gets the project info stored
func GetProjectMeta(projectDir string) (*ProjectMeta, error) {
	contents, err := os.ReadFile(filepath.Join(projectDir, spaceDir, projectMetaFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, err
	}

	projectMeta, err := projectMetaFromBytes(contents)
	if err != nil {
		return nil, err
	}

	return projectMeta, nil
}

func IsProjectInitialized(projectDir string) (bool, error) {
	_, err := os.Stat(filepath.Join(projectDir, spaceDir, projectMetaFile))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

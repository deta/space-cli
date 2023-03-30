package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	spaceVersionPath = ".detaspace/space_latest_version"
)

// ProjectMeta xx
type ProjectMeta struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

// unmarshals data into a ProjectMeta
func projectMetaFromBytes(data []byte) (*ProjectMeta, error) {
	var p ProjectMeta
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type Version struct {
	Version   string `json:"version"`
	UpdatedAt int64  `json:"updatedAt"`
}

func CacheLatestVersion(version string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	spaceDirPath := filepath.Join(home, spaceDir)
	err = os.MkdirAll(spaceDirPath, 0760)
	if err != nil {
		return err
	}

	versionsFilePath := filepath.Join(home, spaceVersionPath)
	content, err := json.Marshal(Version{
		Version:   version,
		UpdatedAt: int64(time.Now().Unix()),
	})
	if err != nil {
		return err
	}

	os.WriteFile(versionsFilePath, content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func GetLatestCachedVersion() (string, time.Time, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", time.Time{}, err
	}

	versionsFilePath := filepath.Join(home, spaceVersionPath)
	content, err := os.ReadFile(versionsFilePath)
	if err != nil {
		return "", time.Time{}, err
	}

	var version Version
	if err = json.Unmarshal(content, &version); err != nil {
		return "", time.Time{}, err
	}

	return version.Version, time.Unix(version.UpdatedAt, 0), nil
}

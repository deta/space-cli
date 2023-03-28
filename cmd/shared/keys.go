package shared

import (
	"fmt"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
)

func GenerateDataKeyIfNotExists(projectID string) (string, error) {
	// check if we have already stored the project key based on the project's id
	projectKey, err := auth.GetProjectKey(projectID)
	if err == nil {
		return projectKey, nil
	}

	listRes, err := Client.ListProjectKeys(projectID)
	if err != nil {
		return "", err
	}

	keyName := findAvailableKey(listRes.Keys, "space cli")

	// create a new project key using the api
	r, err := Client.CreateProjectKey(projectID, &api.CreateProjectKeyRequest{
		Name: keyName,
	})
	if err != nil {
		return "", err
	}

	// store the project key locally
	err = auth.StoreProjectKey(projectID, r.Value)
	if err != nil {
		return "", err
	}

	return r.Value, nil
}

func findAvailableKey(keys []api.ProjectKey, name string) string {
	keyMap := make(map[string]struct{})
	for _, key := range keys {
		keyMap[key.Name] = struct{}{}
	}

	if _, ok := keyMap[name]; !ok {
		return name
	}

	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s (%d)", name, i)
		if _, ok := keyMap[newName]; !ok {
			return newName
		}
	}
}

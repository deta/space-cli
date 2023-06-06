package utils

import (
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/auth"
)

func GenerateDataKeyIfNotExists(projectID string) (string, error) {
	// check if we have already stored the project key based on the project's id
	projectKey, err := auth.GetProjectKey(projectID)
	if err == nil {
		if err := Client.CheckProjectKey(projectKey); err == nil {
			return projectKey, nil
		}
	}

	listRes, err := Client.ListProjectKeys(projectID)
	if err != nil {
		return "", err
	}

	// delete the project key if it exists
	keyName := "space cli"
	for _, key := range listRes.Keys {
		if key.Name == keyName {
			err := Client.DeleteProjectKey(projectID, keyName)
			if err != nil {
				return "", err
			}
			break
		}
	}

	// create a new project key
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

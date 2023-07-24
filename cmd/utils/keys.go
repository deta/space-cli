package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/auth"
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

func GenerateApiKeyIfNotExists(accessToken string, hostname string) (string, error) {
	// check if we have already stored the project key based on the project's id
	apiKey, err := auth.GetApiKey(hostname)
	if err == nil {
		return apiKey, nil
	}

	if !strings.HasSuffix(hostname, "deta.app") {
		return "", fmt.Errorf("custom domain are not supported: %s", hostname)
	}

	parts := strings.Split(hostname, ".")
	instanceAlias := parts[0]

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://deta.space/api/v0/instances?alias=%s", instanceAlias), nil)
	if err != nil {
		return "", err
	}

	if err := Client.AuthenticateRequest(accessToken, req); err != nil {
		return "", err
	}

	resp, err := Client.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch instance: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var listInstanceRes struct {
		Instances []struct {
			ID string `json:"id"`
		} `json:"instances"`
	}
	if err := json.Unmarshal(body, &listInstanceRes); err != nil {
		return "", err
	}

	if len(listInstanceRes.Instances) == 0 {
		return "", fmt.Errorf("no instance found")
	}

	instanceID := listInstanceRes.Instances[0].ID

	createKeyBody, err := json.Marshal(struct {
		Name string `json:"name"`
	}{Name: "space cli"})
	if err != nil {
		return "", err
	}

	createKeyReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://deta.space/api/v0/instances/%s/api_keys", instanceID), bytes.NewReader(createKeyBody))
	if err != nil {
		return "", err
	}

	if err := Client.AuthenticateRequest(accessToken, createKeyReq); err != nil {
		return "", err
	}

	createKeyResp, err := Client.Client.Do(createKeyReq)
	if err != nil {
		return "", err
	}
	defer createKeyResp.Body.Close()

	if createKeyResp.StatusCode != 200 {
		return "", fmt.Errorf("failed to create api key: %s", createKeyResp.Status)
	}

	var createKeyRes struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(createKeyResp.Body).Decode(&createKeyRes); err != nil {
		return "", err
	}

	// store the project key locally
	err = auth.StoreApiKey(hostname, createKeyRes.Value)
	if err != nil {
		return "", err
	}

	return createKeyRes.Value, nil
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

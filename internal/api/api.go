package api

import (
	"encoding/json"
	"fmt"
)

const (
	spaceRoot = "https://d8xfzk.deta.dev" // "https://alpha.deta.space"
	version   = "v0"
)

type CreateProjectRequest struct {
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

type CreateProjectResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

func (c *DetaClient) CreateProject(r *CreateProjectRequest) (*CreateProjectResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/apps", version),
		Method:    "POST",
		NeedsAuth: true,
		Body:      r,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to create project: %v", msg)
	}

	var resp CreateProjectResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create new program: %w", err)
	}
	return &resp, nil
}

type CreateReleaseRequest struct {
	RevisionID  string `json:"revision_id"`
	AppID       string `json:"app_id"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type CreateReleaseResponse struct {
	ID string `json:"id"`
}

func (c *DetaClient) CreateRelease(r *CreateReleaseRequest) (*CreateReleaseResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/promotions", version),
		Method:    "POST",
		NeedsAuth: true,
		Body:      r,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to create release: %v", msg)
	}

	var resp CreateReleaseResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create new release: %w", err)
	}
	return &resp, nil
}

type GetReleaseLogsRequest struct {
	ID string `json:"id"`
}

func (c *DetaClient) GetReleaseLogs(r *GetReleaseLogsRequest, logs chan<- string) error {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/promotions/%s/logs/?follow=true", version, r.ID),
		Method:    "GET",
		NeedsAuth: true,
		LogStream: logs,
	}

	o, err := c.request(i)
	if err != nil {
		return err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return fmt.Errorf("failed to create release: %v", msg)
	}

	return nil
}

type CreateBuildRequest struct {
	AppID string `json:"app_id"`
	Tag   string `json:"tag"`
}

type CreateBuildResponse struct {
	ID string `json:"id"`
}

func (c *DetaClient) CreateBuild(r *CreateBuildRequest) (*CreateBuildResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/builds", version),
		Method:    "GET",
		NeedsAuth: true,
		Body:      r,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to create build request: %v", msg)
	}

	var resp CreateBuildResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create build request: %w", err)
	}
	return &resp, nil
}

// PushManifestRequest push manifest request
type PushManifestRequest struct {
	BuildID  string `json:"build_id"`
	Manifest []byte `json:"manifest"`
}

// PushManifestResponse push manifest response
type PushManifestResponse struct {
	ID string `json:"build_id"`
}

// PushManifest pushes raw manifest file content with an uploadID
func (c *DetaClient) PushManifest(r *PushManifestRequest) (*PushManifestResponse, error) {
	i := &requestInput{
		Root:        spaceRoot,
		Path:        fmt.Sprintf("/%s/builds/%s/manifest", version, r.BuildID),
		Method:      "POST",
		Headers:     make(map[string]string),
		Body:        r.Manifest,
		NeedsAuth:   true,
		ContentType: "text/plain",
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}
	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf("failed to push manifest file, %v", msg)
	}

	var resp PushManifestResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to push manifest file %w", err)
	}

	return &resp, nil
}

// PushCodeRequest push manifest request
type PushCodeRequest struct {
	BuildID    string `json:"build_id"`
	ZippedCode []byte `json:"zipped_code"`
}

// PushManifestResponse push manifest response
type PushCodeResponse struct {
	ID string `json:"build_id"`
}

// PushCode pushes raw manifest file content with an uploadID
func (c *DetaClient) PushCode(r *PushCodeRequest) (*PushCodeResponse, error) {
	i := &requestInput{
		Root:        spaceRoot,
		Path:        fmt.Sprintf("/%s/builds/%s/code", version, r.BuildID),
		Method:      "POST",
		Headers:     make(map[string]string),
		Body:        r.ZippedCode,
		NeedsAuth:   true,
		ContentType: "application/zip",
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}
	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf("failed to push code, %v", msg)
	}

	var resp PushCodeResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to push code, %w", err)
	}

	return &resp, nil
}

type GetBuildLogsRequest struct {
	BuildID string `json:"build_id"`
}

func (c *DetaClient) GetBuildLogs(r *GetBuildLogsRequest, logs chan<- string) error {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/builds/%s/logs/?follow=true", version, r.BuildID),
		Method:    "GET",
		NeedsAuth: true,
		LogStream: logs,
	}

	o, err := c.request(i)
	if err != nil {
		return err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return fmt.Errorf("failed to get build logs: %v", msg)
	}

	return nil
}

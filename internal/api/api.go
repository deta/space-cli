package api

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	spaceRoot = "https://alpha.deta.space/api" // "https://alpha.deta.space"
	//spaceRoot = "http://localhost:9900/api"
	version = "v0"
)

var (
	// ErrProjectNotFound project not found error
	ErrProjectNotFound = fmt.Errorf("project not found")

	// Status
	Complete = "complete"
)

type GetProjectRequest struct {
	ID string `json:"id"`
}

type GetProjectResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

func (c *DetaClient) GetProject(r *GetProjectRequest) (*GetProjectResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/apps/%s", version, r.ID),
		Method:    "GET",
		NeedsAuth: true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status == 401 {
		return nil, ErrProjectNotFound
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to get project: %v", msg)
	}

	var resp GetProjectResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return &resp, nil
}

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

	if o.Status != 201 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to create project: %v", msg)
	}

	var resp CreateProjectResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create new project: %w", err)
	}
	return &resp, nil
}

type CreateReleaseRequest struct {
	RevisionID  string `json:"revision_id"`
	AppID       string `json:"app_id"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Channel     string `json:"channel"`
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

	if o.Status != 202 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
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

func (c *DetaClient) GetReleaseLogs(r *GetReleaseLogsRequest) (io.ReadCloser, error) {
	i := &requestInput{
		Root:             spaceRoot,
		Path:             fmt.Sprintf("/%s/promotions/%s/logs?follow=true", version, r.ID),
		Method:           "GET",
		NeedsAuth:        true,
		ReturnReadCloser: true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to create release: %v", msg)
	}
	return o.BodyReadCloser, nil
}

type GetRevisionsRequest struct {
	ID string `json:"id"`
}

type Revision struct {
	ID        string `json:"id"`
	Tag       string `json:"tag"`
	CreatedAt string `json:"created_at"`
}

type Page struct {
	Size int     `json:"size"`
	Last *string `json:"last"`
}

type fetchRevisionsResponse struct {
	Revisions []Revision `json:"revisions"`
	Page      *Page      `json:"page"`
}

type GetRevisionsResponse struct {
	Revisions []*Revision `json:"revisions"`
}

func (c *DetaClient) GetRevisions(r *GetRevisionsRequest) (*GetRevisionsResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/apps/%s/revisions?limit=5", version, r.ID),
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
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to fetch revisions: %v", msg)
	}

	var fetchResp fetchRevisionsResponse
	err = json.Unmarshal(o.Body, &fetchResp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch revisions: %w", err)
	}

	var revisions []*Revision
	for i := range fetchResp.Revisions {
		revisions = append(revisions, &fetchResp.Revisions[i])
	}

	return &GetRevisionsResponse{Revisions: revisions}, nil
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
		Method:    "POST",
		NeedsAuth: true,
		Body:      r,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 202 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
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

// PushSpacefileRequest push spacefile request
type PushSpacefileRequest struct {
	BuildID  string `json:"build_id"`
	Manifest []byte `json:"manifest"`
}

// PushSpacefileResponse push spacefile response
type PushSpacefileResponse struct {
	ID string `json:"build_id"`
}

// PushSpacefile pushes raw spacefile file content
func (c *DetaClient) PushSpacefile(r *PushSpacefileRequest) (*PushSpacefileResponse, error) {
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
		return nil, fmt.Errorf("failed to push spacefile file, %v", msg)
	}

	var resp PushSpacefileResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to push spacefile file %w", err)
	}

	return &resp, nil
}

// PushIconRequest xx
type PushIconRequest struct {
	BuildID     string `json:"build_id"`
	Icon        []byte `json:"icon"`
	ContentType string `json:"content_type"`
}

// PushIconResponse xx
type PushIconResponse struct {
	ID string `json:"build_id"`
}

// PushIcon pushes icon with an uploadID
func (c *DetaClient) PushIcon(r *PushIconRequest) (*PushIconResponse, error) {
	i := &requestInput{
		Root:        spaceRoot,
		Path:        fmt.Sprintf("/%s/builds/%s/icon", version, r.BuildID),
		Method:      "POST",
		Headers:     make(map[string]string),
		Body:        r.Icon,
		NeedsAuth:   true,
		ContentType: r.ContentType,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if !(o.Status >= 200 && o.Status <= 200) {
		msg := o.Error.Detail
		return nil, fmt.Errorf("failed to push icon, %v", msg)
	}

	var resp PushIconResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to push icon, %w", err)
	}
	return &resp, nil
}

// PushDiscoveryFileRequest xx
type PushDiscoveryFileRequest struct {
	DiscoveryFile []byte `json:"discovery_file"`
	BuildID       string `json:"build_id"`
}

// PushDiscoveryFileResponse xx
type PushDiscoveryFileResponse struct {
	ID string `json:"build_id"`
}

func (c *DetaClient) PushDiscoveryFile(r *PushDiscoveryFileRequest) (*PushDiscoveryFileResponse, error) {
	i := &requestInput{
		Root:        spaceRoot,
		Path:        fmt.Sprintf("/%s/builds/%s/discovery", version, r.BuildID),
		Method:      "POST",
		Headers:     make(map[string]string),
		Body:        r.DiscoveryFile,
		NeedsAuth:   true,
		ContentType: "text/plain",
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if !(o.Status >= 200 && o.Status <= 200) {
		msg := o.Error.Detail
		return nil, fmt.Errorf("failed to push discovery file, %v", msg)
	}

	var resp PushDiscoveryFileResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to push discovery file, %w", err)
	}
	return &resp, nil
}

// PushCodeRequest push spacefile request
type PushCodeRequest struct {
	BuildID    string `json:"build_id"`
	ZippedCode []byte `json:"zipped_code"`
}

// PushSpacefileResponse push spacefile response
type PushCodeResponse struct {
	ID string `json:"build_id"`
}

// PushCode pushes raw spacefile file content with an uploadID
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

func (c *DetaClient) GetBuildLogs(r *GetBuildLogsRequest) (io.ReadCloser, error) {
	i := &requestInput{
		Root:             spaceRoot,
		Path:             fmt.Sprintf("/%s/builds/%s/logs?follow=true", version, r.BuildID),
		Method:           "GET",
		NeedsAuth:        true,
		ReturnReadCloser: true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to get build logs: %v", msg)
	}
	return o.BodyReadCloser, nil
}

type GetBuildRequest struct {
	BuildID string `json:"build_id"`
}

type GetBuildResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (c *DetaClient) GetBuild(r *GetBuildLogsRequest) (*GetBuildResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/builds/%s", version, r.BuildID),
		Method:    "GET",
		NeedsAuth: true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf("failed to get build status, %v", msg)
	}

	var resp GetBuildResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get build status, %w", err)
	}

	return &resp, nil
}

type GetReleasePromotionRequest struct {
	PromotionID string `json:"promotion_id"`
}

type GetReleasePromotionResponse struct {
	ID     string `json:"id" db:"id"`
	Status string `json:"status" db:"status"`
}

func (c *DetaClient) GetReleasePromotion(r *GetReleasePromotionRequest) (*GetReleasePromotionResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/promotions/%s", version, r.PromotionID),
		Method:    "GET",
		NeedsAuth: true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf("failed to get build status, %v", msg)
	}

	var resp GetReleasePromotionResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get build status, %w", err)
	}

	return &resp, nil
}

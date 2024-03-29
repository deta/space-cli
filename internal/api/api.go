package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/deta/space/internal/auth"
	"github.com/deta/space/shared"
)

const (
	version = "v0"
)

var (
	spaceRoot = "https://deta.space/api"

	// ErrProjectNotFound project not found error
	ErrProjectNotFound = errors.New("project not found")
	ErrReleaseNotFound = errors.New("release not found")

	// Complete status
	Complete = "complete"
)

func init() {
	if env, ok := os.LookupEnv("SPACE_ROOT"); ok {
		spaceRoot = env
	}
}

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
		return nil, fmt.Errorf(msg)
	}

	var resp GetProjectResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf(msg)
	}

	var resp CreateProjectResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type CreateReleaseRequest struct {
	RevisionID    string `json:"revision_id"`
	AppID         string `json:"app_id"`
	Version       string `json:"version"`
	ReleaseNotes  string `json:"release_notes"`
	Description   string `json:"description"`
	Channel       string `json:"channel"`
	DiscoveryList bool   `json:"discovery_list"`
	AutoPWA       bool   `json:"auto_pwa"`
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
		return nil, fmt.Errorf(msg)
	}

	var resp CreateReleaseResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf(msg)
	}
	return o.BodyReadCloser, nil
}

type GetRevisionRequest struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
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

type GetRevisionResponse struct {
	Revision *Revision `json:"revision"`
}

func (c *DetaClient) GetRevision(r *GetRevisionRequest) (*GetRevisionResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/apps/%s/revisions/tag/%s", version, r.ID, r.Tag),
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
		return nil, fmt.Errorf(msg)
	}

	var resp GetRevisionResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
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
		Path:      fmt.Sprintf("/%s/apps/%s/revisions?per_page=5", version, r.ID),
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
		return nil, fmt.Errorf(msg)
	}

	var fetchResp fetchRevisionsResponse
	err = json.Unmarshal(o.Body, &fetchResp)
	if err != nil {
		return nil, err
	}

	var revisions []*Revision
	for i := range fetchResp.Revisions {
		revisions = append(revisions, &fetchResp.Revisions[i])
	}

	return &GetRevisionsResponse{Revisions: revisions}, nil
}

type CreateBuildRequest struct {
	AppID        string `json:"app_id"`
	Tag          string `json:"tag"`
	Experimental bool   `json:"experimental"`
	AutoPWA      bool   `json:"auto_pwa"`
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
		return nil, fmt.Errorf(msg)
	}

	var resp CreateBuildResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf(msg)
	}

	var resp PushSpacefileResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
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

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf(msg)
	}

	var resp PushIconResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
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

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf(msg)
	}

	var resp PushDiscoveryFileResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PushCodeRequest push code request
type PushCodeRequest struct {
	BuildID    string `json:"build_id"`
	ZippedCode []byte `json:"zipped_code"`
}

// PushCodeResponse push code response
type PushCodeResponse struct {
	ID string `json:"build_id"`
}

// PushCode pushes raw code
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
		return nil, fmt.Errorf(msg)
	}

	var resp PushCodeResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf(msg)
	}
	return o.BodyReadCloser, nil
}

type GetBuildRequest struct {
	BuildID string `json:"build_id"`
}

type GetBuildResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Tag    string `json:"tag"`
}

func (c *DetaClient) GetBuild(r *GetBuildRequest) (*GetBuildResponse, error) {
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
		return nil, fmt.Errorf(msg)
	}

	var resp GetBuildResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type GetReleasePromotionRequest struct {
	PromotionID string `json:"promotion_id"`
}

type GetReleasePromotionResponse struct {
	ID      string `json:"id" db:"id"`
	Status  string `json:"status" db:"status"`
	Channel string `json:"channel" db:"channel"`
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

	if errors.Is(auth.ErrNoAccessTokenFound, err) {
		return nil, fmt.Errorf("no access token found, please login via space login")
	}

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf(msg)
	}

	var resp GetReleasePromotionResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type MicroPresets struct {
	Environment []*MicroPresetEnv `json:"env"`
}

type MicroPresetEnv struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Value       string `json:"value"`
	Default     string `json:"default,omitempty"`
	DefaultSet  bool   `json:"default_set,omitempty"`
}

type AppInstanceMicro struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Presets *MicroPresets `json:"presets"`
}

type AppInstance struct {
	ID     string              `json:"id"`
	Micros []*AppInstanceMicro `json:"micros,omitempty"`
}

type FetchDevAppInstanceResponse struct {
	Instances []*AppInstance `json:"instances"`
}

func (c *DetaClient) PatchDevAppInstancePresets(instanceID string, micro *AppInstanceMicro) error {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/instances/%s", version, instanceID),
		Method:    "PATCH",
		NeedsAuth: true,
		Body: struct {
			Micros []*AppInstanceMicro `json:"micros"`
		}{Micros: []*AppInstanceMicro{micro}},
	}

	o, err := c.request(i)
	if err != nil {
		return err
	}

	if errors.Is(auth.ErrNoAccessTokenFound, err) {
		return fmt.Errorf("no access token found, please login via space login")
	}

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return fmt.Errorf(msg)
	}

	return nil
}

func (c *DetaClient) GetDevAppInstance(projectID string) (*AppInstance, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/instances?app_id=%s&per_page=1&channel=development", version, projectID),
		Method:    "GET",
		NeedsAuth: true,
		Body:      nil,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if errors.Is(auth.ErrNoAccessTokenFound, err) {
		return nil, fmt.Errorf("no access token found, please login via space login")
	}

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf(msg)
	}

	var fetchResp FetchDevAppInstanceResponse
	err = json.Unmarshal(o.Body, &fetchResp)
	if err != nil {
		return nil, err
	}

	if len(fetchResp.Instances) == 0 {
		return nil, fmt.Errorf("no dev instance found")
	}

	// partially initialized `AppInstance`
	devInstance := fetchResp.Instances[0]

	i = &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/instances/%s", version, devInstance.ID),
		Method:    "GET",
		NeedsAuth: true,
		Body:      nil,
	}

	o, err = c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf(msg)
	}

	err = json.Unmarshal(o.Body, &devInstance)
	if err != nil {
		return nil, err
	}

	return devInstance, nil
}

type GetPromotionRequest struct {
	RevisionID string `json:"revision_id"`
}

type FetchPromotionResponse struct {
	Promotions []GetReleasePromotionResponse `json:"promotions"`
	Page       *Page                         `json:"page"`
}

func (c *DetaClient) GetPromotionByRevision(r *GetPromotionRequest) (*GetReleasePromotionResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/promotions?revision_id=%s&per_page=1", version, r.RevisionID),
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
		return nil, fmt.Errorf(msg)
	}

	var fetchResp FetchPromotionResponse
	err = json.Unmarshal(o.Body, &fetchResp)
	if err != nil {
		return nil, err
	}

	if len(fetchResp.Promotions) == 0 {
		return nil, nil
	}

	promotion := &fetchResp.Promotions[0]
	if promotion.Channel != "development" {
		return nil, fmt.Errorf("no development promotion found")
	}

	return promotion, nil
}

type GetInstallationByReleaseRequest struct {
	ReleaseID string `json:"release_id"`
}

type Installation struct {
	ID         string `json:"id"`
	InstanceID string `json:"instance_id"`
	ReleaseID  string `json:"release_id"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type FetchInstallationsResponse struct {
	Installations []Installation `json:"installations"`
	Page          *Page          `json:"page"`
}

func (c *DetaClient) GetInstallationByRelease(r *GetInstallationByReleaseRequest) (*Installation, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/installations?release_id=%s&per_page=1", version, r.ReleaseID),
		Method:    "GET",
		NeedsAuth: true,
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
		return nil, fmt.Errorf(msg)
	}

	var fetchResp FetchInstallationsResponse
	err = json.Unmarshal(o.Body, &fetchResp)
	if err != nil {
		return nil, err
	}

	var installation *Installation
	if len(fetchResp.Installations) > 0 {
		installation = &fetchResp.Installations[0]
	}

	return installation, nil
}

type GetInstallationRequest struct {
	ID string `json:"id"`
}

func (c *DetaClient) GetInstallation(r *GetInstallationRequest) (*Installation, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/installations/%s", version, r.ID),
		Method:    "GET",
		NeedsAuth: true,
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
		return nil, fmt.Errorf(msg)
	}

	var resp Installation
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type GetInstallationLogsRequest struct {
	ID string `json:"id"`
}

func (c *DetaClient) GetInstallationLogs(r *GetInstallationLogsRequest) (io.ReadCloser, error) {
	i := &requestInput{
		Root:             spaceRoot,
		Path:             fmt.Sprintf("/%s/installations/%s/logs?follow=true", version, r.ID),
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
		return nil, fmt.Errorf(msg)
	}
	return o.BodyReadCloser, nil
}

type GetSpaceRequest struct {
	AccessToken string `json:"access_token"`
}

type GetSpaceResponse struct {
	Name string `json:"name"`
}

func (c *DetaClient) GetSpace(r *GetSpaceRequest) (*GetSpaceResponse, error) {
	i := &requestInput{
		Root:        spaceRoot,
		Path:        fmt.Sprintf("/%s/space", version),
		Method:      "GET",
		NeedsAuth:   true,
		AccessToken: r.AccessToken,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	// unauthorized
	if o.Status == 401 {
		return nil, errors.New("unauthorized")
	}

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf(msg)
	}

	var resp GetSpaceResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type CreateProjectKeyRequest struct {
	Name string `json:"name"`
}

type CreateProjectKeyResponse struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Value     string `json:"value"`
}

func (c *DetaClient) CreateProjectKey(AppID string, r *CreateProjectKeyRequest) (*CreateProjectKeyResponse, error) {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/apps/%s/keys", version, AppID),
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
		return nil, fmt.Errorf(msg)
	}

	var resp CreateProjectKeyResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type ProjectKey struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type ListProjectResponse struct {
	Keys []ProjectKey `json:"keys"`
}

func (c *DetaClient) ListProjectKeys(AppID string) (*ListProjectResponse, error) {
	o, err := c.request(&requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/apps/%s/keys", version, AppID),
		Method:    "GET",
		NeedsAuth: true,
	})
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf(msg)
	}

	var resp ListProjectResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type Release struct {
	ID         string                `json:"id"`
	Tag        string                `json:"tag"`
	ReleasedAt string                `json:"released_at"`
	Discovery  *shared.DiscoveryData `json:"discovery"`
}

func (c *DetaClient) getLatestReleaseByApp(appID string, listed bool) (*Release, error) {
	path := fmt.Sprintf("/%s/releases/latest?app_id=%s", version, appID)
	if listed {
		path = fmt.Sprintf("%s&listed=true", path)
	}
	i := &requestInput{
		Root:      spaceRoot,
		Path:      path,
		Method:    "GET",
		NeedsAuth: true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status == 404 {
		// no release found
		return nil, ErrReleaseNotFound
	}

	if o.Status != 200 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf(msg)
	}

	var release Release
	err = json.Unmarshal(o.Body, &release)
	if err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *DetaClient) GetLatestReleaseByApp(appID string) (*Release, error) {
	return c.getLatestReleaseByApp(appID, false)
}

func (c *DetaClient) GetLatestListedReleaseByApp(appID string) (*Release, error) {
	return c.getLatestReleaseByApp(appID, true)
}

// PushScreenshotRequest xx
type PushScreenshotRequest struct {
	PromotionID string `json:"promotion_id"`
	Index       int    `json:"-"`
	Image       []byte `json:"image"`
	ContentType string `json:"content_type"`
}

// PushScreenshotResponse xx
type PushScreenshotResponse struct {
	ID string `json:"release_id"`
}

// PushScreenshot pushes image
func (c *DetaClient) PushScreenshot(r *PushScreenshotRequest) (*PushScreenshotResponse, error) {

	path := fmt.Sprintf("/%s/promotions/%s/discovery/screenshots/%d", version, r.PromotionID, r.Index)

	i := &requestInput{
		Root:        spaceRoot,
		Path:        path,
		Method:      "POST",
		Headers:     make(map[string]string),
		Body:        r.Image,
		NeedsAuth:   true,
		ContentType: r.ContentType,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if !(o.Status >= 200 && o.Status <= 299) {
		msg := o.Error.Detail
		return nil, fmt.Errorf(msg)
	}

	var resp PushScreenshotResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *DetaClient) StoreDiscoveryData(PromotionID string, r *shared.DiscoveryData) error {
	i := &requestInput{
		Root:      spaceRoot,
		Path:      fmt.Sprintf("/%s/promotions/%s/discovery", version, PromotionID),
		Method:    "POST",
		NeedsAuth: true,
		Body:      r,
	}

	o, err := c.request(i)
	if err != nil {
		return err
	}

	if o.Status != 202 {
		msg := o.Error.Detail
		if msg == "" && len(o.Error.Errors) > 0 {
			msg = o.Error.Errors[0]
		}
		return fmt.Errorf("%v", msg)
	}

	return nil
}

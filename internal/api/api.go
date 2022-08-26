package api

import (
	"fmt"
)

var (
	templatesUrl  = "https://github.com/rohanshiva/actions/archive/refs/heads"
	templatesRepo = "actions"
)

func NewTemplateNotFoundError(template string) error {
	return fmt.Errorf("template %s not found", template)
}

type CreateProjectRequest struct {
	Name  string
	Alias string
}

type CreateProjectResponse struct {
	ProjectId string
}

func (c *DetaClient) CreateProject() *CreateProjectResponse {
	return nil
}

type DownloadTemplateRequest struct {
	Template string
}

type DownloadTemplateResponse struct {
	TemplateFiles  []byte
	TemplatePrefix string
}

func (c *DetaClient) DownloadTemplate(r *DownloadTemplateRequest) (*DownloadTemplateResponse, error) {
	i := &requestInput{
		Root:      templatesUrl,
		Path:      fmt.Sprintf("/%s.zip", r.Template),
		Method:    "GET",
		NeedsAuth: false,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		if o.Status == 404 {
			return nil, NewTemplateNotFoundError(r.Template)
		}

		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to download template files: %v", msg)
	}

	var res DownloadTemplateResponse
	res.TemplateFiles = o.Body
	res.TemplatePrefix = fmt.Sprintf("%s-%s.zip", templatesRepo, r.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to download template files: %v", err)
	}
	return &res, nil
}

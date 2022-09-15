package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/deta/pc-cli/internal/auth"
)

type DetaClient struct {
	Client *http.Client
}

func NewDetaClient() *DetaClient {
	return &DetaClient{
		Client: &http.Client{},
	}
}

type errorResp struct {
	Errors []string `json:"errors,omitempty"`
	Detail string   `json:"message,omitempty"`
}

// requestInput input to Request function
type requestInput struct {
	Root        string
	Path        string
	Method      string
	Headers     map[string]string
	QueryParams map[string]string
	Body        interface{}
	NeedsAuth   bool
	ContentType string
	ReturnReadCloser bool
}

// requestOutput ouput of Request function
type requestOutput struct {
	Status int
	Body   []byte
	BodyReadCloser io.ReadCloser
	Header http.Header
	Error  *errorResp
}

// Request send an http request to the deta api
func (d *DetaClient) request(i *requestInput) (*requestOutput, error) {
	marshalled, _ := i.Body.([]byte)
	if i.Body != nil && i.ContentType == ""{
		// default set content-type to application/json
		i.ContentType = "application/json"
		var err error
		marshalled, err = json.Marshal(&i.Body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(i.Method, fmt.Sprintf("%s%s", i.Root, i.Path), bytes.NewBuffer(marshalled))
	if err != nil {
		return nil, err
	}

	// headers
	if i.ContentType != "" {
		req.Header.Set("Content-type", i.ContentType)
	}
	for k, v := range i.Headers {
		req.Header.Set(k, v)
	}

	// query params
	q := req.URL.Query()
	for k, v := range i.QueryParams {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	if i.NeedsAuth {
		accessToken, err := auth.GetAccessToken()
		if err != nil {
			if errors.Is(err, auth.ErrNoAccessTokenFound) {
				return nil, fmt.Errorf("no access token found, provide access token")
			}
			return nil, fmt.Errorf("failed to authorize")
		}

		//  request timestamp
		now := time.Now().UTC().Unix()
		timestamp := strconv.FormatInt(now, 10)

		// compute signature
		signature, err := auth.CalcSignature(&auth.CalcSignatureInput{
			AccessToken: accessToken,
			HTTPMethod:  i.Method,
			URI:         req.URL.RequestURI(),
			Timestamp:   timestamp,
			ContentType: i.ContentType,
			RawBody:     marshalled,
		})
		if err != nil {
			return nil, err
		}
		// set needed access key auth headers
		req.Header.Set("X-Deta-Timestamp", timestamp)
		req.Header.Set("X-Deta-Signature", signature)
	}

	res, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}


	o := &requestOutput{
		Status: res.StatusCode,
		Header: res.Header,
	}

	if i.ReturnReadCloser && res.StatusCode >= 200 && res.StatusCode <= 299 {
		o.BodyReadCloser = res.Body
		return o, nil
	}

	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		if res.StatusCode != 204 {
			o.Body = b
		}
		return o, nil
	}

	var er errorResp
	if res.StatusCode == 413 {
		er.Detail = "Request entity too large"
		o.Error = &er
		return o, nil
	}
	if res.StatusCode == 502 {
		er.Detail = "Internal server error"
		o.Error = &er
		return o, nil
	}
	err = json.Unmarshal(b, &er)
	if err != nil {
		return nil, err
	}
	o.Error = &er
	return o, nil
}

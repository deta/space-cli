package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	"github.com/deta/space/shared"
)

const (
	actionEndpoint      = "/__space/actions"
	clientBaseEndpoint  = "/__space/v0/base"
	clientDriveEndpoint = "/__space/v0/drive"
)

const (
	baseHost  = "database.deta.sh"
	driveHost = "drive.deta.sh"
)

type ProxyEndpoint struct {
	Micro shared.Micro
	Port  int
}

type ActionMeta struct {
	Actions []DevAction `json:"actions"`
}

type DevAction struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	Input  any    `json:"input"`
	Output string `json:"output"`
}

type ProxyAction struct {
	Url           string `json:"-"`
	InstanceID    string `json:"instance_id"`
	InstanceAlias string `json:"instance_alias"`
	AppName       string `json:"app_name"`
	Name          string `json:"name"`
	Title         string `json:"title"`
	Channel       string `json:"channel"`
	Version       string `json:"version"`
	Input         any    `json:"input,omitempty"`
	Output        string `json:"output,omitempty"`
}

type ReverseProxy struct {
	appID         string
	appName       string
	instanceAlias string
	prefixToProxy map[string]*httputil.ReverseProxy
	actionMap     map[string]ProxyAction
	projectKey    string
	client        *http.Client
}

func NewReverseProxy(projectKey string, appID string, appName string, instanceAlias string) *ReverseProxy {
	return &ReverseProxy{
		appID:         appID,
		appName:       appName,
		instanceAlias: instanceAlias,
		prefixToProxy: make(map[string]*httputil.ReverseProxy),
		actionMap:     make(map[string]ProxyAction),
		projectKey:    projectKey,
		client:        &http.Client{},
	}
}

func (p *ReverseProxy) AddMicro(micro *shared.Micro, port int) (int, error) {
	prefix := extractPrefix(micro.Path)
	p.prefixToProxy[prefix] = httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", port),
	})

	if !micro.ProvideActions {
		return 0, nil
	}

	res, err := http.Get(fmt.Sprintf("http://localhost:%d%s", port, actionEndpoint))
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	var actionMeta ActionMeta
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&actionMeta); err != nil {
		return 0, err
	}

	for _, devAction := range actionMeta.Actions {
		if devAction.Output == "" {
			devAction.Output = "@deta/raw"
		}

		var target string
		if strings.HasPrefix(devAction.Path, "/") {
			target = fmt.Sprintf("http://localhost:%d%s", port, devAction.Path)
		} else {
			target = fmt.Sprintf("http://localhost:%d/%s", port, devAction.Path)
		}

		p.actionMap[devAction.Name] = ProxyAction{
			Url:           target,
			InstanceID:    p.appID,
			InstanceAlias: p.instanceAlias,
			AppName:       p.appName,
			Name:          devAction.Name,
			Title:         devAction.Title,
			Channel:       "local",
			Version:       "dev",
			Input:         devAction.Input,
			Output:        devAction.Output,
		}
	}
	return len(actionMeta.Actions), nil
}

func extractPrefix(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return "/" + parts[1]
	}

	return "/"
}

func (p *ReverseProxy) ServeClientSDKAuth(targetHost string, w http.ResponseWriter, r *http.Request) {
	newURL := *r.URL
	newURL.Host = targetHost
	newURL.Scheme = "https"
	newURL.Path = regexp.MustCompile("^/__space/v0/(drive|base)/v1/[^/]+").
		ReplaceAllString(newURL.Path, "/v1/"+strings.Split(p.projectKey, "_")[0])

	newReq, err := http.NewRequest(r.Method, newURL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newReq.Header.Add("X-API-Key", p.projectKey)
	newReq.Header.Add("Content-Type", "application/json")
	resp, err := p.client.Do(newReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, clientBaseEndpoint) {
		p.ServeClientSDKAuth(baseHost, w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, clientDriveEndpoint) {
		p.ServeClientSDKAuth(driveHost, w, r)
		return
	}

	if r.URL.Path == actionEndpoint {
		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "https://deta.space")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusOK)
			return
		case http.MethodGet:
			var actions = make([]ProxyAction, 0, len(p.actionMap))
			for _, action := range p.actionMap {
				actions = append(actions, action)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "https://deta.space")
			w.Header().Set("Access-Control-Allow-Headers", "*")

			encoder := json.NewEncoder(w)
			if err := encoder.Encode(actions); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			return
		}
	}

	if strings.HasPrefix(r.URL.Path, actionEndpoint) {
		actionName := strings.TrimPrefix(r.URL.Path, actionEndpoint+"/")
		action, ok := p.actionMap[actionName]
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "https://deta.space")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusOK)
			return
		case http.MethodGet:
			action, ok := p.actionMap[actionName]
			if !ok {
				http.NotFound(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "https://deta.space")
			w.Header().Set("Access-Control-Allow-Headers", "*")

			encoder := json.NewEncoder(w)
			if err := encoder.Encode(action); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			return
		case http.MethodPost:
			resp, err := http.Post(action.Url, "application/json", r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var data any
			if err := json.Unmarshal(body, &data); err != nil {
				data = string(body)
			}

			payload := map[string]interface{}{
				"type": action.Output,
				"data": data,
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "https://deta.space")
			w.Header().Set("Access-Control-Allow-Headers", "*")

			encoder := json.NewEncoder(w)
			if err := encoder.Encode(payload); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	prefix := extractPrefix(r.URL.Path)
	if proxy, ok := p.prefixToProxy[prefix]; ok {
		if prefix != "/" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		}
		proxy.ServeHTTP(w, r)
		return
	}

	fallback, ok := p.prefixToProxy["/"]
	if ok {
		fallback.ServeHTTP(w, r)
		return
	}
}

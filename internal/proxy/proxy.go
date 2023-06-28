package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/deta/space/shared"
)

const (
	actionEndpoint = "/__space/actions"
)

type ProxyEndpoint struct {
	Micro shared.Micro
	Port  int
}

type ActionMeta struct {
	Actions []Action `json:"actions"`
}

type Action struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	Input  any    `json:"input"`
	Output string `json:"output"`
}

type ReverseProxy struct {
	appID         string
	appName       string
	instanceAlias string
	prefixToProxy map[string]*httputil.ReverseProxy
	actionToProxy map[string]*httputil.ReverseProxy
	actionMap     map[string]map[string]any
}

func NewReverseProxy(appID string, appName string, instanceAlias string) *ReverseProxy {
	return &ReverseProxy{
		appID:         appID,
		appName:       appName,
		instanceAlias: instanceAlias,
		prefixToProxy: make(map[string]*httputil.ReverseProxy),
		actionToProxy: make(map[string]*httputil.ReverseProxy),
		actionMap:     make(map[string]map[string]any),
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
		p.actionToProxy[devAction.Name] = httputil.NewSingleHostReverseProxy(&url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", port),
			Path:   devAction.Path,
		})

		action := map[string]any{
			"instance_id":    p.appID,
			"instance_alias": p.instanceAlias,
			"app_name":       p.appName,
			"name":           devAction.Name,
			"title":          devAction.Title,
			"channel":        "local",
			"version":        "dev",
			"output":         devAction.Output,
		}

		if devAction.Output != "" {
			action["output"] = devAction.Output
		} else {
			action["output"] = "@deta/raw"
		}

		if devAction.Input != nil {
			action["input"] = devAction.Input
		}

		p.actionMap[devAction.Name] = action
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

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == actionEndpoint {
		var actions = make([]map[string]any, 0, len(p.actionMap))
		for _, action := range p.actionMap {
			actions = append(actions, action)
		}

		encoder := json.NewEncoder(w)
		if err := encoder.Encode(actions); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if strings.HasPrefix(r.URL.Path, actionEndpoint) {
		actionName := strings.TrimPrefix(r.URL.Path, actionEndpoint+"/")
		proxy, ok := p.actionToProxy[actionName]
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			action, ok := p.actionMap[actionName]
			if !ok {
				http.NotFound(w, r)
				return
			}

			encoder := json.NewEncoder(w)
			if err := encoder.Encode(action); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		case http.MethodPost:
			r.URL.Path = ""
			proxy.ServeHTTP(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
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

	http.NotFound(w, r)
}

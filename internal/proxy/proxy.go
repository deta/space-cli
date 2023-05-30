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
	Name  string `json:"name"`
	Title string `json:"title"`
	Path  string `json:"path"`
	Input any    `json:"input"`
}

type ReverseProxy struct {
	prefixToProxy map[string]*httputil.ReverseProxy
	actionToProxy map[string]*httputil.ReverseProxy
	actions       []Action
}

func NewReverseProxy() *ReverseProxy {
	prefixToProxy := make(map[string]*httputil.ReverseProxy)
	actionToProxy := make(map[string]*httputil.ReverseProxy)

	return &ReverseProxy{
		prefixToProxy: prefixToProxy,
		actionToProxy: actionToProxy,
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

	for _, action := range actionMeta.Actions {
		p.actionToProxy[action.Name] = httputil.NewSingleHostReverseProxy(&url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", port),
			Path:   action.Path,
		})

		p.actions = append(p.actions, action)
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
		var actions = make([]map[string]any, len(p.actions))
		for i, action := range p.actions {
			actions[i] = map[string]any{
				"instance_alias": "dev",
				"instance_id":    "dev",
				"app_name":       "dev",
				"name":           action.Name,
				"title":          action.Title,
				"channel":        "local",
				"version":        "dev",
			}

			if action.Input != nil {
				actions[i]["input"] = action.Input
			}
		}

		encoder := json.NewEncoder(w)
		if err := encoder.Encode(actions); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if strings.HasPrefix(r.URL.Path, actionEndpoint) {
		actionName := strings.TrimPrefix(r.URL.Path, actionEndpoint+"/")
		if proxy, ok := p.actionToProxy[actionName]; ok {
			r.URL.Path = ""
			proxy.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
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

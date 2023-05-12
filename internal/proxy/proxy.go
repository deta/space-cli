package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	actionsPath = "/__space/actions"
)

type ActionMeta struct {
	Actions []MicroAction `json:"actions"`
}

type MicroAction struct {
	Name   string        `json:"name"`
	Path   string        `json:"path"`
	Inputs []ActionInput `json:"inputs,omitempty"`
}

type ActionInput struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
}

type AppAction struct {
	Name    string        `json:"name"`
	AppName string        `json:"app_name"`
	Inputs  []ActionInput `json:"inputs,omitempty"`
}

type ReverseProxy struct {
	prefixToProxy map[string]*httputil.ReverseProxy
	actionToProxy map[string]*httputil.ReverseProxy
	actions       []AppAction
}

func NewReverseProxy() *ReverseProxy {
	return &ReverseProxy{
		prefixToProxy: make(map[string]*httputil.ReverseProxy),
		actionToProxy: make(map[string]*httputil.ReverseProxy),
	}
}

func extractPrefix(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return "/" + parts[1]
	}

	return "/"
}

func (p *ReverseProxy) AddRoute(prefix string, target *url.URL) {
	p.prefixToProxy[prefix] = httputil.NewSingleHostReverseProxy(target)
}

func (p *ReverseProxy) ExtractActions(target *url.URL) error {
	actionUrl := url.URL{
		Scheme: target.Scheme,
		Host:   target.Host,
		Path:   "/__space/actions",
	}

	res, err := http.Get(actionUrl.String())
	if err != nil {
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil
	}

	var meta ActionMeta
	if err := json.NewDecoder(res.Body).Decode(&meta); err != nil {
		return nil
	}

	for _, action := range meta.Actions {
		actionUrl := url.URL{
			Scheme: target.Scheme,
			Host:   target.Host,
			Path:   action.Path,
		}
		p.actionToProxy[action.Name] = httputil.NewSingleHostReverseProxy(&actionUrl)
		p.actions = append(p.actions, AppAction{
			Name:    action.Name,
			AppName: "dev",
			Inputs:  action.Inputs,
		})
	}

	return nil
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == actionsPath || r.URL.Path == actionsPath+"/" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		json.NewEncoder(w).Encode(p.actions)
		return
	}

	if strings.HasPrefix(r.URL.Path, actionsPath) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		actionName := strings.TrimPrefix(r.URL.Path, actionsPath+"/")
		if proxy, ok := p.actionToProxy[actionName]; ok {
			r.URL.Path = "/"
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

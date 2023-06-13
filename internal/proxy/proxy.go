package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
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
	prefixToUrl map[string]*url.URL
	actionToUrl map[string]*url.URL
	actionMap   map[string]map[string]any
	proxy       *httputil.ReverseProxy
}

func NewReverseProxy() *ReverseProxy {
	prefixToUrl := make(map[string]*url.URL)
	actionToUrl := make(map[string]*url.URL)
	actionMap := make(map[string]map[string]any)

	return &ReverseProxy{
		prefixToUrl: prefixToUrl,
		actionToUrl: actionToUrl,
		actionMap:   actionMap,
		proxy: &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				for action, url := range actionToUrl {
					if r.URL.Path == fmt.Sprintf("/__space/actions/%s", action) {
						r.URL.Scheme = url.Scheme
						r.URL.Host = url.Host
						r.URL.Path = url.Path
						return
					}
				}

				requestPath := r.URL.Path

				// if request is coming from a page, use the page's path as the request path
				if referer := r.Header.Get("Referer"); referer != "" {
					refererUrl, err := url.Parse(referer)
					if err != nil {
						return
					}
					requestPath = path.Join(refererUrl.Path, r.URL.Path)
				}

				prefix := extractPrefix(requestPath)
				if target, ok := prefixToUrl[prefix]; ok {
					r.URL.Scheme = target.Scheme
					r.URL.Host = target.Host
					r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
					if r.URL.Path == "" {
						r.URL.Path = "/"
					}

					return
				}

				if fallback, ok := prefixToUrl["/"]; ok {
					r.URL.Scheme = fallback.Scheme
					r.URL.Host = fallback.Host
					return
				}
			},
		},
	}
}

func (p *ReverseProxy) AddMicro(micro *shared.Micro, port int) (int, error) {
	prefix := extractPrefix(micro.Path)
	p.prefixToUrl[prefix] = &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", port),
	}

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
		p.actionToUrl[action.Name] = &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", port),
		}
		p.actionMap[action.Name] = map[string]any{
			"instance_alias": "dev",
			"instance_id":    "dev",
			"app_name":       "dev",
			"name":           action.Name,
			"title":          action.Title,
			"channel":        "local",
			"version":        "dev",
		}

		if action.Input != nil {
			p.actionMap[action.Name]["input"] = action.Input
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

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.proxy == nil {
		http.Error(w, "proxy not initialized", http.StatusInternalServerError)
	}

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

	p.proxy.ServeHTTP(w, r)
}

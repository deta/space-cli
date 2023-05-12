package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type ProxyRoute struct {
	Prefix string   `json:"prefix"`
	Target *url.URL `json:"target"`
}

type ReverseProxy struct {
	prefixToProxy map[string]*httputil.ReverseProxy
	actionToProxy map[string]*httputil.ReverseProxy
}

func NewReverseProxy(routes []ProxyRoute, actions map[string]*url.URL) *ReverseProxy {
	prefixToProxy := make(map[string]*httputil.ReverseProxy)
	for _, route := range routes {
		proxy := httputil.NewSingleHostReverseProxy(route.Target)
		prefixToProxy[route.Prefix] = proxy
	}

	actionToProxy := make(map[string]*httputil.ReverseProxy)
	for actionName, action := range actions {
		proxy := httputil.NewSingleHostReverseProxy(action)
		actionToProxy[actionName] = proxy
	}

	return &ReverseProxy{
		prefixToProxy: prefixToProxy,
		actionToProxy: actionToProxy,
	}
}

func extractPrefix(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return "/" + parts[1]
	}

	return "/"
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/__space/actions") {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		actionName := strings.TrimPrefix(r.URL.Path, "/__space/actions/")
		if proxy, ok := p.actionToProxy[actionName]; ok {
			r.URL.Path = "/"
			proxy.ServeHTTP(w, r)
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

	http.NotFound(w, r)
}

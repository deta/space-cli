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
}

func NewReverseProxy(routes []ProxyRoute) *ReverseProxy {
	prefixToProxy := make(map[string]*httputil.ReverseProxy)
	for _, route := range routes {
		proxy := httputil.NewSingleHostReverseProxy(route.Target)
		prefixToProxy[route.Prefix] = proxy
	}

	return &ReverseProxy{
		prefixToProxy: prefixToProxy,
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

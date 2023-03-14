package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
)

type ProxyRoute struct {
	Prefix string   `json:"prefix"`
	Target *url.URL `json:"target"`
}

type ReverseProxy struct {
	prefixToProxy map[string]*httputil.ReverseProxy
	prefixes      []string
}

func NewReverseProxy(routes []ProxyRoute) *ReverseProxy {
	prefixToProxy := make(map[string]*httputil.ReverseProxy)
	prefixes := make([]string, 0)
	for _, route := range routes {
		proxy := httputil.NewSingleHostReverseProxy(route.Target)
		prefixToProxy[route.Prefix] = proxy
		prefixes = append(prefixes, route.Prefix)
	}

	// Sort the prefixes by length, so that we can match the longest prefix first
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})

	return &ReverseProxy{
		prefixToProxy: prefixToProxy,
		prefixes:      prefixes,
	}
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, prefix := range p.prefixes {
		if strings.HasPrefix(r.URL.Path, prefix) {
			if prefix != "/" {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			}

			proxy := p.prefixToProxy[prefix]
			proxy.ServeHTTP(w, r)
			return
		}
	}

	http.NotFound(w, r)
}

package router

import (
	"errors"
	"net/http"
	"strings"
	"sync"
)

type Router struct {
	mu      sync.RWMutex
	Hosts   map[string]Host
}

type Host struct {
	Domain  string          `json:"domain"`
	Label   string          `json:"label"`
	Path    string          `json:"path"`
	Prefix  string          `json:"prefix"`
	handler http.Handler
}

func NewRouter() *Router {
	return &Router{
		Hosts:  make(map[string]Host),
	}
}

func (router *Router) Register(label string, domain string, path string, prefix string, handler http.Handler) {
	// Key in a host by its label
	router.Hosts[label] = Host{
        Domain: domain,
        Label: label,
        Path: path,
        Prefix: prefix,
        handler: handler,
    }
}

func (router *Router) Lookup(path string) (host Host, err error) {
	router.mu.RLock()
	defer router.mu.RUnlock()

	// Extract the prefix from the given path
	split := strings.Split(path, "/")
	if len(split) >= 2 {
		prefix := split[1]
		host = router.Hosts[prefix]
		if host.handler != nil {
			return host, nil
		}
	}

	err = errors.New("404 Not Found")
	return Host{}, err
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Verify request signature
    r.ParseForm()
    f := r.Form

    // authorization := router

    valid := Authenticate(f.Get("digest"), f.Get("key"), f.Get("now"), r.URL.Path, r.Method)
    if valid != true {
        w.WriteHeader(http.StatusUnauthorized)
		return
    }

	// Fetch host by the given path
	host, err := router.Lookup(r.URL.Path)
	if err != nil {
	    http.NotFound(w, r)
		return
	}

	// Build new path removing prefix
	split := strings.Split(r.URL.Path, "/")
	r.URL.Path = "/" + strings.Join(split[2:], "/")

	// Assign target host header
	r.Host = host.Domain

	// Assign handler
	handler := host.handler

	// Serve request
	handler.ServeHTTP(w, r)
}

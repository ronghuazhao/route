package route

import (
	"net/http"
	"strings"
	"sync"
)

type Router struct {
	mu    sync.RWMutex
	hosts map[string]Host
}

type Host struct {
	domain  string
	handler http.Handler
}

func NewRouter() *Router {
	return &Router{hosts: make(map[string]Host)}
}

func (router *Router) Register(label string, domain string, prefix string, handler http.Handler) {
	// Key in a host by its label
	router.hosts[label] = Host{domain: domain, handler: handler}
}

func (router *Router) Lookup(path string) (host Host) {
	router.mu.RLock()
	defer router.mu.RUnlock()

	// Extract the prefix from the given path
	split := strings.Split(path, "/")
	prefix := split[1]

	// Find the host from its prefix
	host = router.hosts[prefix]

	return host
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Fetch host by the given path
	host := router.Lookup(r.URL.Path)

	// Build new path removing prefix
	split := strings.Split(r.URL.Path, "/")
	r.URL.Path = "/" + strings.Join(split[2:], "/")

	// Assign target host header
	r.Host = host.domain

	// Assign handler
	handler := host.handler

	// Send event to central log
	Log("route", "request.start", r.Method+" "+r.Host+r.URL.String())

	// Serve request
	handler.ServeHTTP(w, r)
}

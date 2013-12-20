package main

import (
	"net/http"
	"strings"
	"sync"
)

type Mux struct {
	mu    sync.RWMutex
	hosts map[string]Host
}

type Host struct {
	domain  string
	handler http.Handler
}

func NewMux() *Mux {
	return &Mux{hosts: make(map[string]Host)}
}

func (mux *Mux) Register(label string, domain string, prefix string, handler http.Handler) {
	// Key in a host by its label
	mux.hosts[label] = Host{domain: domain, handler: handler}
}

func (mux *Mux) Lookup(path string) (host Host) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	// Extract the prefix from the given path
	split := strings.Split(path, "/")
	prefix := split[1]

	// Find the host from its prefix
	host = mux.hosts[prefix]

	return host
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Fetch host by the given path
	host := mux.Lookup(r.URL.Path)

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

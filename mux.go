package main

import (
    "github.com/Shopify/sarama"
    "net/http"
    "strings"
    "sync"
)

type Mux struct {
    mu         sync.RWMutex
    routes     map[string]http.Handler
}

type muxEntry struct {
    prefix  bool
    handler http.Handler
}

func NewMux() *Mux {
    return &Mux{routes: make(map[string]http.Handler)}
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Pass event to central log
    mux.log(r.Method + " " + r.URL.Path)

    handler := mux.lookup(r.URL.Path)

    // Build new path removing prefix
    split := strings.Split(r.URL.Path, "/")
    r.URL.Path = "/" + strings.Join(split[2:], "/")

    handler.ServeHTTP(w, r)
}

func (mux *Mux) lookup(path string) (handler http.Handler) {
    mux.mu.RLock()
    defer mux.mu.RUnlock()

    split := strings.Split(path, "/")

    route := mux.routes[split[1]]

    return route
}

func (mux *Mux) Handle(path string, label string, handler http.Handler) {
    mux.routes[label] = handler
}

func (mux *Mux) log(path string) {
    var err error
    err = log.QueueMessage("route", sarama.StringEncoder("request.start"), sarama.StringEncoder(path))
    if err != nil {
        panic(err)
    }
}

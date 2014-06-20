// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

/*
Package router defines an HTTP router that authenticates requests, performs the request, and returns the result to a client.
*/
package router

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"api.umn.edu/route/cache"
	"api.umn.edu/route/interfaces"
	"api.umn.edu/route/logger"
	"api.umn.edu/route/util"
	"code.google.com/p/goprotobuf/proto"
	zmq "github.com/alecthomas/gozmq"
)

// Struct representing a route
type Route struct {
	Description string `json:"description"`
	Id          string `json:"id"`
	Domain      string `json:"domain"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	handler     http.Handler
}

/* request
Path    string `json:"path"`
*/

// Struct representing a router
type Router struct {
	mutex  sync.RWMutex
	Routes map[string]Route
}

// Global variables
var logging *logger.Logger
var local_cache *cache.Cache

// Constants
const timeout string = "2s"

func init() {
	// Create logging instance
	logging = logger.NewLogger("router", logger.Console)

	// Connect to cache
	var err error
	local_cache, err = cache.NewCache(cache.Redis)
	if err != nil {
		logging.Log("internal", "route.error", "failed to bind to redis", "[fg-red]")
		os.Exit(1)
	}
}

// NewRouter initializes a router instance.
func NewRouter() *Router {
	return &Router{
		Routes: make(map[string]Route),
	}
}

// Register accepts a new route to handle
func (router *Router) Register(route Route) {
	// Lock router to add a new host
	router.mutex.RLock()
	defer router.mutex.RUnlock()

	// Create reverse proxy
	url, _ := url.Parse("http://" + route.Domain + route.Path)

	route.handler = httputil.NewSingleHostReverseProxy(url)

	// Add host keyed by ID
	router.Routes[route.Id] = route

	// Store route in cache
	cache_route := map[string]string{
		"description": route.Description,
		"id":          route.Id,
		"domain":      route.Domain,
		"name":        route.Name,
		"path":        route.Path,
	}

	local_cache.Set(fmt.Sprintf("route:%s", route.Id), cache_route)

	// Set up publisher context
	context, err := zmq.NewContext()
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ context", "[fg-red]")
		return
	}

	// Automatically close context when finished
	defer context.Close()

	// Set up socket
	s, err := context.NewSocket(zmq.REQ)
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ socket", "[fg-red]")
		return
	}

	// If we close the socket, don't wait for any existing requests
	s.SetLinger(0)

	// Parse a timeout value
	rcv_timeout, err := time.ParseDuration(timeout)
	if err != nil {
		logging.Log("internal", "route.error", "invalid timeout specified", "[fg-red]")
		return
	}

	// Set the timeout for receiving messages
	// Note that this must be done before connecting to an address
	s.SetRcvTimeout(rcv_timeout)

	// Automatically close socket when finished
	defer s.Close()

	// Connect to the publisher
	s.Connect(util.GetenvDefault("PUBLISH_BIND", "tcp://127.0.0.1:6667"))
	logging.Log("internal", "route.start", "event publisher started", "[fg-blue]")

	// Build a message to send to the store
	message := &interfaces.Route{
		Description: proto.String(route.Description),
		Id:          proto.String(route.Id),
		Domain:      proto.String(route.Domain),
		Name:        proto.String(route.Name),
		Path:        proto.String(route.Path),
	}

	// Marshal message into protobuf
	data, err := proto.Marshal(message)
	if err != nil {
		logging.Log("internal", "router.error", "unable to marshal message", "[fg-red]")
		return
	}

	// Broadcast message
	s.SendMultipart([][]byte{[]byte("route"), data}, 0)

	// Receive acknowledgement
	_, err = s.RecvMultipart(0)

	// Check if message sending failed
	if err != nil {
		logging.Log("internal", "route.error", "storage connection timed out", "[fg-red]")
		logging.Log("internal", "route.error", "operating without store", "[fg-red]")

		// Clean up after failure
		s.Close()
		context.Close()
		return
	}

	logging.Log("internal", "route.publish", "route published", "[fg-blue]")
}

// Lookup retrieves a host from a request path and registers it as a HTTP reverse proxy with the router.
// It then returns the host that was registered.
func (router *Router) Route(request string) (host Route, err error) {
	// Lock router to register and lookup host
	router.mutex.RLock()
	defer router.mutex.RUnlock()

	// Extract the prefix from the given path
	path := strings.Split(request, "/")
	if len(path) > 1 {
		id := path[1]
		host := router.Routes[id]
		return host, nil
	}

	err = errors.New("Route not found")
	return Route{}, err
}

// ServeHTTP receives requests, authenticates them, and then reverse-proxies the request to the backend API.
// It then returns the resource to the client.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse form values from request
	r.ParseForm()
	form := r.Form

	// Get request values
	digest := form.Get("digest")
	public_key := form.Get("key")
	now := form.Get("now")
	path := r.URL.Path
	method := r.Method

	/* Load private key from cache */
	keypair, _ := local_cache.Get(fmt.Sprintf("key:%s", public_key))

	var valid bool
	if len(keypair) == 2 {
		private_key := keypair[1]

		// Authenticate the request
		valid = Authenticate(digest, public_key, private_key, now, path, method)
	} else {
		valid = false
	}

	// Abort if the message is not properly authenticated
	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Fetch host by the given path
	host, err := router.Route(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Build new path and remove prefix
	split := strings.Split(r.URL.Path, "/")
	r.URL.Path = "/" + strings.Join(split[2:], "/")

	r.Host = host.Domain

	// Serve request
	host.handler.ServeHTTP(w, r)
}

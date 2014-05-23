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
	"api.umn.edu/route/logger"
	"api.umn.edu/route/util"
	"api.umn.edu/store/interfaces"
	"code.google.com/p/goprotobuf/proto"
	zmq "github.com/alecthomas/gozmq"
)

// Struct representing a route
type Route struct {
	Name        string
	Description string
	Endpoint    string
}

// Struct representing a router
type Router struct {
	mutex sync.RWMutex
	Hosts map[string]Host
}

// Struct representing a host
type Host struct {
	Domain  string `json:"domain"`
	Label   string `json:"label"`
	Path    string `json:"path"`
	Prefix  string `json:"prefix"`
	handler http.Handler
}

/* Global variables */
var logging *logger.Logger
var local_cache *cache.Cache

/* Constants */
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
		Hosts: make(map[string]Host),
	}
}

// Register accepts parameters for a new host to route to.
func (router *Router) Register(label string, domain string, path string, prefix string, handler http.Handler) {
	// Lock router to add a new host
	router.mutex.RLock()
	defer router.mutex.RUnlock()

	// Add host keyed by label
	router.Hosts[label] = Host{
		Domain:  domain,
		Label:   label,
		Path:    path,
		Prefix:  prefix,
		handler: handler,
	}

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

	// Store route in cache
	route := map[string]string{
		"Label":  label,
		"Domain": domain,
		"Path":   path,
		"Prefix": prefix,
	}

	local_cache.Set(fmt.Sprintf("route:%s", label), route)

	// Build a message to send to the store
	message := &interfaces.Route{
		Do:     interfaces.DO_UPDATE.Enum(),
		Id:     proto.String("0"),
		Label:  proto.String(label),
		Path:   proto.String(path),
		Prefix: proto.String(prefix),
		Domain: proto.String(domain),
	}

	// Marshal message into protobuf
	data, err := proto.Marshal(message)
	if err != nil {
		logging.Log("internal", "router.error", "unable to marshal message", "[fg-red]")
		return
	}

	// Broadcast message
	s.SendMultipart([][]byte{[]byte("route"), data}, 0)

	// Block and receive acknowledgement
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
func (router *Router) Lookup(path string) (host Host, err error) {
	// Lock router to register and lookup host
	router.mutex.RLock()
	defer router.mutex.RUnlock()

	// Extract the prefix from the given path
	split := strings.Split(path, "/")
	if len(split) < 2 {
		err = errors.New("Route not found")
		return Host{}, err
	}
	prefix := split[1]

	// Load route from cache
	var route struct {
		Domain string
		Label  string
		Prefix string
		Path   string
	}

	_, err = local_cache.Get(fmt.Sprintf("route:%s", prefix))
	if err != nil {
		return
	}

	domain := route.Domain
	label := route.Label
	routeprefix := route.Prefix
	routepath := route.Path

	// Create reverse proxy
	url, _ := url.Parse(path)
	proxy := httputil.NewSingleHostReverseProxy(url)

	/* Register route */
	router.Register(label, domain, routepath, routeprefix, proxy)

	host = router.Hosts[prefix]
	return host, nil
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
	fmt.Printf("%s\n", keypair)
	private_key := keypair[1]

	// Authenticate the request
	valid := Authenticate(digest, public_key, private_key, now, path, method)
	fmt.Println(valid)
	fmt.Println(public_key)
	fmt.Println(private_key)

	// Abort if the message is not properly authenticated
	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Fetch host by the given path
	host, err := router.Lookup(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Build new path and remove prefix
	split := strings.Split(r.URL.Path, "/")
	r.URL.Path = "/" + strings.Join(split[2:], "/")

	// Assign target host header
	r.Host = host.Domain

	// Assign handler
	handler := host.handler

	// Serve request
	handler.ServeHTTP(w, r)
}

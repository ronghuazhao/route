// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

// Package router defines an HTTP router that authenticates requests, performs the request, and returns the result to a client.
package router

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"api.umn.edu/mailman"
	"api.umn.edu/route/cache"
	"api.umn.edu/route/interfaces"
	"code.google.com/p/goprotobuf/proto"
)

// Types
type Route struct {
	Description string `json:"description"`
	Id          string `json:"id"`
	Domain      string `json:"domain"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	handler     http.Handler
}

type Router struct {
	Routes map[string]Route
	Topics []string
	lock   *sync.RWMutex
	mail   *mailman.Mailman
	cache  *cache.Cache
}

// Constants
const timeout string = "0.5s"
const publishTopic string = "route"

// NewRouter initializes a router instance
func NewRouter(sub, req string) *Router {
	// Connect to messaging
	mail, err := mailman.NewMailman(sub, req, timeout)
	if err != nil {
		panic(err)
	}

	// Connect to cache
	cursor, err := cache.NewCache(cache.Redis)
	if err != nil {
		panic(err)
	}

	return &Router{
		Routes: make(map[string]Route),
		mail:   mail,
		cache:  cursor,
	}
}

func (router *Router) Listen() {
	// Start event listener
	fmt.Println("starting listener")
	router.mail.Listen([]string{"auth"}, router.handle)
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

	router.cache.Set(fmt.Sprintf("route:%s", route.Id), cache_route)

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
		panic(err)
	}

	// Broadcast message
	response := router.mail.Send(mailman.CreateAction, "route", data)

	// Check if message sending failed
	if response != mailman.OkResponse {
		return
	}
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
	keypair, _ := router.cache.Get(fmt.Sprintf("key:%s", public_key))

	// TODO: If that fails, attempt to fetch the key from the central service

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

// The handle function is the callback for the event listener and handles
// incoming messages on the given topic.
func (router *Router) handle(topic string, code int, payload []byte) {
	fmt.Println("handling")
	fmt.Println(topic)
	switch {
	case topic == "auth":
		/* Auth message */
		data := &interfaces.Auth{}

		/* Extract message into structure */
		err := proto.Unmarshal(payload, data)
		if err != nil {
			panic(err)
		}

		/* Store in appropriate collection based on topic */
		public_key := data.GetPublicKey()
		private_key := data.GetPrivateKey()
		fmt.Println(public_key)
		fmt.Println(private_key)

		payload := map[string]string{
			public_key: private_key,
		}

		router.cache.Set(fmt.Sprintf("key:%s", public_key), payload)
	}
}

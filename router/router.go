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

	// Topics to listen to events on
	topics := []string{"auth"}

	// Create lock for modifying the router
	lock := &sync.RWMutex{}

	return &Router{
		Routes: make(map[string]Route),
		Topics: topics,
		cache:  cursor,
		lock:   lock,
		mail:   mail,
	}
}

func (router *Router) Listen() {
	// Start event listener
	router.mail.Listen(router.Topics, router.handle)
}

// Register accepts a new route to handle
func (router *Router) Register(route Route) {
	// Lock router to add a new host
	router.mutex.RLock()
	defer router.mutex.RUnlock()

	// Create reverse proxy
	url, _ := url.Parse("http://" + route.Domain + route.Path)
	route.handler = httputil.NewSingleHostReverseProxy(url)

	// Lock router to add a new host
	router.lock.Lock()
	router.Routes[route.Id] = route
	router.lock.Unlock()

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
		return err
	}

	// Broadcast message
	response := router.mail.Send(mailman.CreateAction, publishTopic, data)

	// Check if message sending failed
	if response != mailman.OkResponse {
		return errors.New("Could not publish new route")
	}

	return nil
}

// Route retrieves a host from a request path.
func (router *Router) Route(request string) (host Route, err error) {

	// Extract the prefix from the given path
	path := strings.Split(request, "/")
	if len(path) > 1 {
		id := path[1]

		// Lock router to lookup host
		router.lock.RLock()
		host := router.Routes[id]
		router.lock.RUnlock()

		return host, nil
	}

	return Route{}, errors.New("Route not found")
}

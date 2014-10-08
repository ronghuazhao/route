// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

/*
Route is an authenticated API router. It manages directing incoming requests to backend APIs.
It mandates authenticating every request and integrates with a keyserver to validate API keys.
*/
package main

import (
	"net/http"
	"runtime"

	"api.umn.edu/route/logger"
	"api.umn.edu/route/router"
	"api.umn.edu/route/util"
	"code.google.com/p/gcfg"
)

// Host config file structure
type Config struct {
	Host map[string]*struct {
		Description string
		Id          string
		Domain      string
		Name        string
		Path        string
	}
}

// Global variables
var logging *logger.Logger
var routing *router.Router

func init() {
	// Create logging instance
	logging = logger.NewLogger("route", logger.Console)

	// Create routing instance
	routing = router.NewRouter()
}

func main() {
	// Use all available cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Create core API handler
	core := NewApi("/core/v1", routing)

	// Read in host file
	var hosts Config
	gcfg.ReadFileInto(&hosts, util.GetenvDefault("HOSTS_FILE", "hosts.conf"))

	// Create route handlers from config
	for host, conf := range hosts.Host {
		route := &router.Route{
			Description: "",
			Id:          conf.Id,
			Domain:      conf.Domain,
			Name:        host,
			Path:        conf.Path,
		}

		// Request registration with the router
		routing.Register(*route)
	}

	// Listen for events from the central store
	go routing.Listen()
	logging.Log("internal", "route.start", "event listener started", "[fg-blue]")

	// Listen for external API requests
	go http.ListenAndServe(util.GetenvDefault("ROUTER_BIND", ":8080"), routing)
	logging.Log("internal", "route.start", "router started", "[fg-blue]")

	// Listen for internal API requests
	go http.ListenAndServe(util.GetenvDefault("COREAPI_BIND", ":8081"), core)
	logging.Log("internal", "route.start", "core api started", "[fg-blue]")

	<-make(chan int)
}

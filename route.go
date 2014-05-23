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
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"

	"api.umn.edu/route/cache"
	"api.umn.edu/route/events"
	"api.umn.edu/route/logger"
	"api.umn.edu/route/router"
	"api.umn.edu/route/util"
	"code.google.com/p/gcfg"
)

/* Host config file struct to unmarshal into */
type Config struct {
	Host map[string]*struct {
		Label string
	}
}

/* Global variables */
var logging *logger.Logger
var routing *router.Router
var local_cache *cache.Cache

/* Constants */
var topics = []string{"auth", "route"}

func init() {
	// Create logging instance
	logging = logger.NewLogger("route", logger.Console)

	// Create routing instance
	routing = router.NewRouter()

	// Connect to cache
	var err error
	local_cache, err = cache.NewCache(cache.Redis)
	if err != nil {
		logging.Log("internal", "route.error", "failed to bind to redis", "[fg-red]")
		os.Exit(1)
	}
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
		url, _ := url.Parse(host)
		domain := url.Host
		label := conf.Label
		prefix := "/" + label
		path := url.String()
		proxy := httputil.NewSingleHostReverseProxy(url)

		// Request registration with the router
		routing.Register(label, domain, path, prefix, proxy)
	}

	// Listen for events from the central store
	go events.Listen()
	logging.Log("internal", "route.start", "event listener started", "[fg-blue]")

	// Listen for external API requests
	go http.ListenAndServe(util.GetenvDefault("ROUTER_BIND", ":8080"), routing)
	logging.Log("internal", "route.start", "router started", "[fg-blue]")

	// Listen for internal API requests
	go http.ListenAndServe(util.GetenvDefault("COREAPI_BIND", ":8081"), core)
	logging.Log("internal", "route.start", "core api started", "[fg-blue]")

	<-make(chan int)
}

// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

/*
Route is an authenticated API router. It manages directing incoming requests to backend APIs.
It mandates authenticating every request and integrates with a keyserver to validate API keys.
*/
package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"

	"api.umn.edu/route/interfaces"
	"api.umn.edu/route/logger"
	"api.umn.edu/route/router"
	"api.umn.edu/route/util"
	"code.google.com/p/gcfg"
	"code.google.com/p/goprotobuf/proto"
	zmq "github.com/alecthomas/gozmq"
	"github.com/garyburd/redigo/redis"
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
var cache redis.Conn

/* Constants */
var topics = []string{"auth", "route"}

func init() {
	// Create logging instance
	logging = logger.NewLogger("route", logger.Console)

	// Create routing instance
	routing = router.NewRouter()

	// Connect to cache
	var err error
	cache, err = redis.Dial("tcp", util.GetenvDefault("REDIS_BIND", ":6379"))
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
	go StoreListener(cache)
	logging.Log("internal", "route.start", "storage listener started", "[fg-blue]")

	// Listen for external API requests
	go http.ListenAndServe(util.GetenvDefault("ROUTER_BIND", ":8080"), routing)
	logging.Log("internal", "route.start", "router started", "[fg-blue]")

	// Listen for internal API requests
	go http.ListenAndServe(util.GetenvDefault("COREAPI_BIND", ":8081"), core)
	logging.Log("internal", "route.start", "core api started", "[fg-blue]")

	<-make(chan int)
}

func StoreListener(store redis.Conn) {
	// Create event listener context
	context, err := zmq.NewContext()
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ context", "[fg-red]")
		return
	}

	// Automatically close context when finished
	defer context.Close()

	// Create event listener socket
	s, err := context.NewSocket(zmq.SUB)
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ socket", "[fg-red]")
		return
	}

	// Attempt to bind to storage server
	s.Connect(util.GetenvDefault("EVENT_BIND", "tcp://127.0.0.1:6666"))

	// Automatically close socket when finished
	defer s.Close()

	// Subscribe to event topics
	for _, topic := range topics {
		s.SetSubscribe(topic)
	}

	// Listen
	for {
		// Receive multipart message
		message, _ := s.RecvMultipart(0)

		// Extract message parts
		topic := message[0]
		body := message[1]

		switch {
		case bytes.Equal(topic, []byte(topics[0])):
			// Receive auth message
			auth := &interfaces.Auth{}

			// Unmarshal message into struct
			err := proto.Unmarshal(body, auth)

			if err != nil {
				logging.Log("internal", "route.error", "failed unmarshal message", "[fg-red]")
				return
			}

			// Build a message to be cached
			public_key := auth.GetPublicKey()
			private_key := auth.GetPrivateKey()

			// Build a key to associate the private key against
			cache_key := fmt.Sprintf("key:%s", public_key)

			// Save message to cache
			cache.Do("set", cache_key, private_key)

			logging.Log("internal", "route.event", "key added to cache", "[fg-green]")
		}
	}
}

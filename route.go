package main

import (
	"code.google.com/p/gcfg"
	"github.umn.edu/umnapi/route.git/logger"
	"github.umn.edu/umnapi/route.git/router"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
)

// Host config file structure
type Config struct {
	Host map[string]*struct {
		Label string
	}
}

func main() {

	// Use all cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Bootstrap modules
	logger := logger.NewLogger("route", logger.Kafka)
	router := router.NewRouter(logger)

	// Create API handler
    api := NewApi("/api/v1", router, logger)

	// Read in host file
	var hosts Config
	gcfg.ReadFileInto(&hosts, "hosts.conf")

	// Create route handlers
	for host, conf := range hosts.Host {
		url, _ := url.Parse(host)

		domain := url.Host
		label := conf.Label

		proxy := httputil.NewSingleHostReverseProxy(url)
		prefix := "/" + label
		path := url.String()

		router.Register(label, domain, path, prefix, proxy)
	}

	go http.ListenAndServe(":8080", router)
	logger.Log("internal", "route.start", "router started", "[fg-blue]")

    go http.ListenAndServe(":8081", api)
	logger.Log("internal", "route.start", "api started", "[fg-blue]")

    <- make(chan int)
}

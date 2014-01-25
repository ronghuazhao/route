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

var logging *logger.Logger
var routing *router.Router

func init() {
    // Initiate logger
    logging = logger.NewLogger("route", logger.Console)
	routing = router.NewRouter()
}

func main() {

	// Use all cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Bootstrap modules

	// Create API handler
	api := NewApi("/api/v1", routing)

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

		routing.Register(label, domain, path, prefix, proxy)
	}

	go http.ListenAndServe(":8080", routing)
	logging.Log("internal", "route.start", "router started", "[fg-blue]")

	go http.ListenAndServe(":8081", api)
	logging.Log("internal", "route.start", "api started", "[fg-blue]")

	<-make(chan int)
}

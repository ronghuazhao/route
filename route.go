package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"

	"api.umn.edu/route/events"
	"api.umn.edu/route/logger"
	"api.umn.edu/route/router"
	"api.umn.edu/route/util"
	"code.google.com/p/gcfg"
)

/* Host config file structure */
type Config struct {
	Host map[string]*struct {
		Label string
	}
}

var logging *logger.Logger
var routing *router.Router

var topics = []string{"auth", "route"}

func init() {
	/* Create logger */
	logging = logger.NewLogger("route", logger.Console)

	/* Create router */
	routing = router.NewRouter()
}

func main() {
	/* Use all cores */
	runtime.GOMAXPROCS(runtime.NumCPU())

	/* Create core API handler */
	core := NewApi("/core/v1", routing)

	/* Read in host file */
	var hosts Config
	gcfg.ReadFileInto(&hosts, util.GetenvDefault("HOSTS_FILE", "hosts.conf"))

	/* Create route handlers */
	for host, conf := range hosts.Host {
		url, _ := url.Parse(host)

		domain := url.Host
		label := conf.Label

		proxy := httputil.NewSingleHostReverseProxy(url)
		prefix := "/" + label
		path := url.String()

		routing.Register(label, domain, path, prefix, proxy)
	}

	/* Listen for store events */
	go events.Listen()
	logging.Log("internal", "route.start", "event listener started", "[fg-blue]")

	/* Start router */
	go http.ListenAndServe(util.GetenvDefault("ROUTER_BIND", ":8080"), routing)
	logging.Log("internal", "route.start", "router started", "[fg-blue]")

	/* Start core */
	go http.ListenAndServe(util.GetenvDefault("COREAPI_BIND", ":8081"), core)
	logging.Log("internal", "route.start", "core api started", "[fg-blue]")

	<-make(chan int)
}

package main

import (
	"bytes"
	"code.google.com/p/gcfg"
	"code.google.com/p/goprotobuf/proto"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/garyburd/redigo/redis"
	"github.umn.edu/umnapi/route.git/interfaces"
	"github.umn.edu/umnapi/route.git/logger"
	"github.umn.edu/umnapi/route.git/router"
	"github.umn.edu/umnapi/route.git/util"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
)

/* Host config file structure */
type Config struct {
	Host map[string]*struct {
		Label string
	}
}

var logging *logger.Logger
var routing *router.Router
var cache redis.Conn

var topics = []string{"auth", "route"}

func init() {
	/* Create logger */
	logging = logger.NewLogger("route", logger.Console)

	/* Create router */
	routing = router.NewRouter()

	/* Connect to cache */
	var err error
	cache, err = redis.Dial("tcp", util.GetenvDefault("REDIS_BIND", ":6379"))
	if err != nil {
		logging.Log("internal", "route.error", "failed to bind to redis", "[fg-red]")
		os.Exit(1)
	}
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
	go eventListener(cache)
	logging.Log("internal", "route.start", "event listener started", "[fg-blue]")

	/* Start router */
	go http.ListenAndServe(util.GetenvDefault("ROUTER_BIND", ":8080"), routing)
	logging.Log("internal", "route.start", "router started", "[fg-blue]")

	/* Start core */
	go http.ListenAndServe(util.GetenvDefault("COREAPI_BIND", ":8081"), core)
	logging.Log("internal", "route.start", "core api started", "[fg-blue]")

	<-make(chan int)
}

func eventListener(store redis.Conn) {
	/* Set up event listener */
	context, err := zmq.NewContext()
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ context", "[fg-red]")
		return
	}

	defer context.Close()

	s, err := context.NewSocket(zmq.SUB)
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ socket", "[fg-red]")
		return
	}

	s.Connect(util.GetenvDefault("EVENT_BIND", "tcp://127.0.0.1:6666"))

	defer s.Close()

	/* Subscribe to event topics */
	for _, topic := range topics {
		s.SetSubscribe(topic)
	}

	/* Listen loop */
	for {
		message, _ := s.RecvMultipart(0)

		/* Extract message parts */
		topic := message[0]
		body := message[1]

		switch {
		case bytes.Equal(topic, []byte(topics[0])):
			/* Auth message */
			data := &interfaces.Auth{}

			/* Extract message into structure */
			err := proto.Unmarshal(body, data)

			if err != nil {
				logging.Log("internal", "route.error", "failed demarshall message", "[fg-red]")
				return
			}

			/* Store in appropriate collection based on topic */
			public_key := data.GetPublicKey()
			private_key := data.GetPrivateKey()
			cache_key := fmt.Sprintf("key:%s", public_key)
			cache.Do("set", cache_key, private_key)

			logging.Log("internal", "route.event", "key added to cache", "[fg-green]")
		}
	}
}

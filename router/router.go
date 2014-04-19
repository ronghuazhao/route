package router

import (
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/garyburd/redigo/redis"
	"github.umn.edu/umnapi/route.git/interfaces"
	"github.umn.edu/umnapi/route.git/logger"
	"github.umn.edu/umnapi/route.git/util"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
)

type Route struct {
	Name        string
	Description string
	Endpoint    string
}

type Router struct {
	mutex sync.RWMutex
	Hosts map[string]Host
	store redis.Conn
}

type Host struct {
	Domain  string `json:"domain"`
	Label   string `json:"label"`
	Path    string `json:"path"`
	Prefix  string `json:"prefix"`
	handler http.Handler
}

var logging *logger.Logger
var cache redis.Conn

func init() {
	/* Create logger */
	logging = logger.NewLogger("router", logger.Console)

	/* Connect to cache */
	var err error
	cache, err = redis.Dial("tcp", util.GetenvDefault("REDIS_BIND", ":6379"))
	if err != nil {
		logging.Log("internal", "route.error", "failed to bind to redis", "[fg-red]")
		os.Exit(1)
	}
}

func NewRouter() *Router {
	return &Router{
		Hosts: make(map[string]Host),
	}
}

func (router *Router) Register(label string, domain string, path string, prefix string, handler http.Handler) {
	/* Add host by label */
	router.Hosts[label] = Host{
		Domain:  domain,
		Label:   label,
		Path:    path,
		Prefix:  prefix,
		handler: handler,
	}

	/* Set up publisher */
	context, err := zmq.NewContext()
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ context", "[fg-red]")
		return
	}

	defer context.Close()

	s, err := context.NewSocket(zmq.REQ)
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ socket", "[fg-red]")
		return
	}

	defer s.Close()

	s.Connect(util.GetenvDefault("PUBLISH_BIND", "tcp://127.0.0.1:6667"))
	logging.Log("internal", "route.start", "event publisher started", "[fg-blue]")

	/* Store route in cache */
	cache_key := fmt.Sprintf("route:%s", label)
	cache.Do("HMSET", cache_key, "label", label, "domain", domain, "path", path, "prefix", prefix)

	/* Broadcast route */
	message := &interfaces.Route{
		Do:     interfaces.DO_UPDATE.Enum(),
		Id:     proto.String("0"),
		Label:  proto.String(label),
		Path:   proto.String(path),
		Prefix: proto.String(prefix),
		Domain: proto.String(domain),
	}

	/* Marshal structure */
	data, err := proto.Marshal(message)
	if err != nil {
		logging.Log("internal", "router.error", "unable to marshall message", "[fg-red]")
		return
	}

	/* Broadcast */
	logging.Log("internal", "route.publish", "publishing route", "[fg-blue]")
	s.SendMultipart([][]byte{[]byte("route"), data}, 0)
	s.RecvMultipart(0)
}

func (router *Router) Lookup(path string) (host Host, err error) {
	router.mutex.RLock()
	defer router.mutex.RUnlock()

	/* Extract the prefix from the given path */
	split := strings.Split(path, "/")
	if len(split) < 2 {
		err = errors.New("Route not found")
		return Host{}, err
	}

	prefix := split[1]

	/* Load route from cache */
	cache_key := fmt.Sprintf("route:%s", prefix)

	/* Create route handler */
	domain, _ := redis.String(cache.Do("HGET", cache_key, "domain"))
	label, _ := redis.String(cache.Do("HGET", cache_key, "label"))
	routeprefix, _ := redis.String(cache.Do("HGET", cache_key, "prefix"))
	path, _ = redis.String(cache.Do("HGET", cache_key, "path"))

	/* Create reverse proxy */
	url, _ := url.Parse(path)
	proxy := httputil.NewSingleHostReverseProxy(url)

	/* Register route */
	router.Register(label, domain, path, routeprefix, proxy)

	host = router.Hosts[prefix]
	return host, nil
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/* Verify request signature */
	r.ParseForm()
	form := r.Form

	/* Get request values */
	digest := form.Get("digest")
	public_key := form.Get("key")
	now := form.Get("now")
	path := r.URL.Path
	method := r.Method

	/* Load private key from cache */
	var private_key string
	cache_key := fmt.Sprintf("key:%s", public_key)
	private_key, _ = redis.String(cache.Do("get", cache_key))

	valid := Authenticate(digest, public_key, private_key, now, path, method)

	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	/* Fetch host by the given path */
	host, err := router.Lookup(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	/* Build new path removing prefix */
	split := strings.Split(r.URL.Path, "/")
	r.URL.Path = "/" + strings.Join(split[2:], "/")

	/* Assign target host header */
	r.Host = host.Domain

	/* Assign handler */
	handler := host.handler

	/* Serve request */
	handler.ServeHTTP(w, r)
}

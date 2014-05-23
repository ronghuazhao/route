package router

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"api.umn.edu/route/cache"
	"api.umn.edu/route/interfaces"
	"api.umn.edu/route/logger"
	"api.umn.edu/route/util"
	"code.google.com/p/goprotobuf/proto"
	zmq "github.com/alecthomas/gozmq"
)

const timeout string = "2s"

type Route struct {
	Name        string
	Description string
	Endpoint    string
}

type Router struct {
	mutex sync.RWMutex
	Hosts map[string]Host
}

type Host struct {
	Domain  string `json:"domain"`
	Label   string `json:"label"`
	Path    string `json:"path"`
	Prefix  string `json:"prefix"`
	handler http.Handler
}

var logging *logger.Logger
var local_cache *cache.Cache

func init() {
	/* Create logger */
	logging = logger.NewLogger("router", logger.Console)

	/* Connect to cache */
	var err error
	local_cache, err = cache.NewCache(cache.Redis)
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

func (router *Router) Register(Label string, Domain string, Path string, Prefix string, handler http.Handler) {
	/* Add host by label */
	router.Hosts[Label] = Host{
		Domain:  Domain,
		Label:   Label,
		Path:    Path,
		Prefix:  Prefix,
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

	if err != nil {
		logging.Log("internal", "route.error", "invalid socket timeout specified", "[fg-red]")
		return
	}

	s.SetLinger(0)

	rcv_timeout, err := time.ParseDuration(timeout)
	s.SetRcvTimeout(rcv_timeout)
	defer s.Close()

	s.Connect(util.GetenvDefault("PUBLISH_BIND", "tcp://127.0.0.1:6667"))
	logging.Log("internal", "route.start", "event publisher started", "[fg-blue]")

	// Store route in cache
	route := map[string]string{
		"Label":  Label,
		"Domain": Domain,
		"Path":   Path,
		"Prefix": Prefix,
	}

	local_cache.Set(fmt.Sprintf("route:%s", Label), route)

	/* Broadcast route */
	message := &interfaces.Route{
		Do:     interfaces.DO_UPDATE.Enum(),
		Id:     proto.String("0"),
		Label:  proto.String(Label),
		Path:   proto.String(Path),
		Prefix: proto.String(Prefix),
		Domain: proto.String(Domain),
	}

	/* Marshal structure */
	data, err := proto.Marshal(message)
	if err != nil {
		logging.Log("internal", "router.error", "unable to marshall message", "[fg-red]")
		return
	}

	/* Broadcast */
	s.SendMultipart([][]byte{[]byte("route"), data}, 0)

	_, err = s.RecvMultipart(0)
	if err != nil {
		logging.Log("internal", "route.error", "storage connection timed out", "[fg-red]")
		logging.Log("internal", "route.error", "operating without store", "[fg-red]")
		s.Close()
		return
	}

	logging.Log("internal", "route.publish", "route published", "[fg-blue]")
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

	// Load route from cache
	var route struct {
		Domain string
		Label  string
		Prefix string
		Path   string
	}

	_, err = local_cache.Get(fmt.Sprintf("route:%s", prefix))
	if err != nil {
		return
	}

	domain := route.Domain
	label := route.Label
	routeprefix := route.Prefix
	routepath := route.Path

	/* Create reverse proxy */
	url, _ := url.Parse(path)
	proxy := httputil.NewSingleHostReverseProxy(url)

	/* Register route */
	router.Register(label, domain, routepath, routeprefix, proxy)

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
	keypair, _ := local_cache.Get(fmt.Sprintf("key:%s", public_key))
	fmt.Printf("%s\n", keypair)
	private_key := keypair[1]

	valid := Authenticate(digest, public_key, private_key, now, path, method)
	fmt.Println(valid)
	fmt.Println(public_key)
	fmt.Println(private_key)

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

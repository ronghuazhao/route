package router

import (
    "errors"
    "fmt"
    "net/http"
    "strings"
    "sync"
    "net/url"
    "net/http/httputil"
    "github.com/garyburd/redigo/redis"
    zmq "github.com/alecthomas/gozmq"
    "code.google.com/p/goprotobuf/proto"
    "github.umn.edu/umnapi/route.git/interfaces"
    "github.umn.edu/umnapi/route.git/logger"
)

type Route struct {
    Name string
    Description string
    Endpoint string
}

type Router struct {
    mu          sync.RWMutex
    Hosts       map[string]Host
    store       redis.Conn
    logging     *logger.Logger
}

type Host struct {
    Domain  string          `json:"domain"`
    Label   string          `json:"label"`
    Path    string          `json:"path"`
    Prefix  string          `json:"prefix"`
    handler http.Handler
}

func NewRouter(store redis.Conn, logging *logger.Logger) *Router {
    return &Router{
	Hosts:  make(map[string]Host),
	store: store,
	logging: logging,
    }
}

func (router *Router) Register(label string, domain string, path string, prefix string, handler http.Handler) {
    // Key in a host by its label
    router.Hosts[label] = Host{
	Domain: domain,
	Label: label,
	Path: path,
	Prefix: prefix,
	handler: handler,
    }

    // Create zmq context
    ctx, _ := zmq.NewContext()
    defer ctx.Close()

    // Create and open zmq socket
    sock, _ := ctx.NewSocket(zmq.REQ)
    sock.Connect("tcp://127.0.0.1:6667")
    defer sock.Close()

    store := router.store;

    // store using the label as the key since that is what we use to build our urls.
    redis_key := fmt.Sprintf("route:%s", label)
    store.Do("HMSET", redis_key, "label", redis_key, "domain", domain, "path", path, "prefix", prefix)

    // Create protobuf object
    message := &interfaces.Route {
	Do: interfaces.DO_UPDATE.Enum(),
	Id: proto.String("0"),
	Label: proto.String(label),
	Path: proto.String(path),
	Prefix: proto.String(prefix),
	Domain: proto.String(domain),
    }

    data, err := proto.Marshal(message)
    if err != nil {
    	router.logging.Log("internal", "router.register", "unable to marshal message", "[fg-red]")
    	return
    }

    sock.SendMultipart([][]byte{[]byte("route"), data}, 0)
    sock.RecvMultipart(0)
}

func (router *Router) Lookup(path string) (host Host, err error) {
    router.mu.RLock()
    defer router.mu.RUnlock()

    // Extract the prefix from the given path
    split := strings.Split(path, "/")
    if len(split) >= 2 {
	prefix := split[1]

	// Load routes from database
	//route := Route{}
	redis_key := fmt.Sprintf("route:%s", prefix)

	// Create route handler
	domain, _ := redis.String(router.store.Do("HGET", redis_key, "domain"))
	label, _ := redis.String(router.store.Do("HGET", redis_key, "label"))
	routeprefix, _ := redis.String(router.store.Do("HGET", redis_key, "prefix"))
	path, _ := redis.String(router.store.Do("HGET", redis_key, "path"))

	url, _ := url.Parse(path)

        proxy := httputil.NewSingleHostReverseProxy(url)

	router.Register(label, domain, path, routeprefix, proxy)

	host = router.Hosts[prefix]
	return host, nil
    }

    err = errors.New("404 Not Found")
    return Host{}, err
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Verify request signature
    r.ParseForm()
    f := r.Form

    // Get request variables
    digest := f.Get("digest")
    public_key := f.Get("key")
    now := f.Get("now")
    path := r.URL.Path
    method := r.Method

    // Get private key
    var private_key string
    redis_key := fmt.Sprintf("keys:%s", public_key)
    store := router.store
    private_key, _ = redis.String(store.Do("get", redis_key))
    /*  router.keyStore.QueryRow("SELECT private_key FROM keystore WHERE public_key=?", f.Get("key")).Scan(&private_key) */

    valid := Authenticate(digest, public_key, private_key, now, path, method)
    // authorized := Authorize()
    if valid != true {
	w.WriteHeader(http.StatusUnauthorized)
	return
    }

    // Fetch host by the given path
    host, err := router.Lookup(r.URL.Path)
    if err != nil {
	http.NotFound(w, r)
	return
    }

    // Build new path removing prefix
    split := strings.Split(r.URL.Path, "/")
    r.URL.Path = "/" + strings.Join(split[2:], "/")

    // Assign target host header
    r.Host = host.Domain

    // Assign handler
    handler := host.handler

    // Serve request
    handler.ServeHTTP(w, r)
}

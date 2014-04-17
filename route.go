package main

import (
    "bytes"
    "github.umn.edu/umnapi/route.git/logger"
    "github.umn.edu/umnapi/route.git/router"
    "github.umn.edu/umnapi/route.git/interfaces"
    "code.google.com/p/goprotobuf/proto"
    "net/http"
    "net/http/httputil"
    "net/url"
    "code.google.com/p/gcfg"
    "runtime"
    "github.com/garyburd/redigo/redis"
    "fmt"
    zmq "github.com/alecthomas/gozmq"
)

// Host config file structure
type Config struct {
    Host map[string]*struct {
	Label string
    }
}

var logging *logger.Logger
var routing *router.Router
var store redis.Conn

func init() {
    // Initiate logger
    store, _ = redis.Dial("tcp", ":6379")
    logging = logger.NewLogger("route", logger.Console)
    routing = router.NewRouter(store, logging)
}

func main() {

    // Use all cores
    runtime.GOMAXPROCS(runtime.NumCPU())

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

    go zmqListen(store)

    // Start router
    go http.ListenAndServe(":8080", routing)
    logging.Log("internal", "route.start", "router started", "[fg-blue]")

    // Start router API
    go http.ListenAndServe(":8081", api)
    logging.Log("internal", "route.start", "api started", "[fg-blue]")

    <-make(chan int)
}

func zmqListen(store redis.Conn) {
    println("started listening on 6666")
    ctx, _ := zmq.NewContext()
    sock, _ := ctx.NewSocket(zmq.SUB)
    sock.Connect("tcp://localhost:6666")
    defer sock.Close()
    sock.SetSubscribe("auth")
    sock.SetSubscribe("route")
    for {
	message, _ := sock.RecvMultipart(0)

	topic := message[0]
	raw := message[1]

	err := error(nil);

	if bytes.Equal(topic, []byte("auth")) {
	    //store auth
	    data := &interfaces.Auth{}
	    err = proto.Unmarshal(raw, data)
	    // 2) store in appropriate colleciton/table based on topic
	    logging.Log("internal", "route.new_keypair", fmt.Sprintf("{%s}", data), "[fg-green]")
	    public_key := data.GetPublicKey()
	    private_key := data.GetPrivateKey()
	    redis_key := fmt.Sprintf("key:%s", public_key)
	    println(redis_key)
	    store.Do("set", redis_key, private_key)
	    test, _ := redis.String(store.Do("get", redis_key))
	    println(test)
	}

	if err != nil {
	    println("demarshalling error")
	}
    }
}

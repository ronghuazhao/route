package router

import (
	"errors"
	"fmt"
	"database/sql"
	"github.umn.edu/umnapi/route.git/logger"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"strings"
	"sync"
	"crypto/hmac"
	"crypto/sha256"
)

type Router struct {
	mu      sync.RWMutex
	Hosts   map[string]Host
	logger  *logger.Logger
}

type Host struct {
	Domain  string          `json:"domain"`
	Label   string          `json:"label"`
	Path    string          `json:"path"`
	Prefix  string          `json:"prefix"`
	handler http.Handler
}

func NewRouter(logger *logger.Logger) *Router {
	return &Router{
		Hosts:  make(map[string]Host),
		logger: logger,
	}
}

func (router *Router) Verify(other string, user string, time string, path string, method string) bool {
    db, _ := sql.Open("sqlite3", "/Users/ben/Code/api-auth/db/development.sqlite3")

    var key string
    rows, _ := db.Query("SELECT private_key FROM keystore WHERE public_key=?", user)
    for rows.Next() {
        rows.Scan(&key)
    }

    mac := hmac.New(sha256.New, []byte(key))
    signature := user + time + path + method
    mac.Write([]byte(signature))
    sum := mac.Sum(nil)

    local := fmt.Sprintf("%x", []byte(sum))

    return hmac.Equal([]byte(local), []byte(other))
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
}

func (router *Router) Lookup(path string) (host Host, err error) {
	router.mu.RLock()
	defer router.mu.RUnlock()

	// Extract the prefix from the given path
	split := strings.Split(path, "/")
	if len(split) >= 2 {
		prefix := split[1]
		host = router.Hosts[prefix]
		if host.handler != nil {
			return host, nil
		}
	}

	err = errors.New("404 Not Found")
	return Host{}, err
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Verify request signature
    r.ParseForm()
    f := r.Form

    valid := router.Verify(f.Get("digest"), f.Get("key"), f.Get("now"), r.URL.Path, r.Method)
    if valid != true {
        message := "401 Unauthorized"
		router.logger.Log("auth", "request.failure", message, "[fg-red]")
		return
    } else {
        message := fmt.Sprintf("Authorized user %s", f.Get("key"))
		router.logger.Log("auth", "request.success", message, "[fg-green]")
    }

	// Fetch host by the given path
	host, err := router.Lookup(r.URL.Path)
	if err != nil {
		message := fmt.Sprintf("%s %s (%s)", r.Method, r.URL.String(), err)
		router.logger.Log("route", "request.failure", message, "[fg-red]")
		return
	}

	// Build new path removing prefix
	split := strings.Split(r.URL.Path, "/")
	r.URL.Path = "/" + strings.Join(split[2:], "/")

	// Assign target host header
	r.Host = host.Domain

	// Assign handler
	handler := host.handler

	// Send event to central log
	message := fmt.Sprintf("%s %s%s (200 OK)", r.Method, r.Host, r.URL.String())
	router.logger.Log("route", "request.start", message, "[fg-green]")

	// Serve request
	handler.ServeHTTP(w, r)
}

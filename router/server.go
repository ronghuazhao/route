package router

import (
	"fmt"
	"net/http"
	"strings"
)

// ServeHTTP receives requests, authenticates them, and then reverse-proxies the request to the backend API.
// It then returns the resource to the client.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse form values from request
	r.ParseForm()
	form := r.Form

	// Get request values
	digest := form.Get("digest")
	public_key := form.Get("key")
	now := form.Get("now")
	path := r.URL.Path
	method := r.Method

	// Load private key from cache
	keypair, _ := router.cache.Get(fmt.Sprintf("key:%s", public_key))

	// TODO: If no key is found, attempt to fetch the key from the central service

	if len(keypair) != 2 {
		http.Error(w, "Invalid key lookup", http.StatusInternalServerError)
		return
	}

	// Authenticate the request
	private_key := keypair[1]
	valid := Authenticate(digest, public_key, private_key, now, path, method)

	// Abort if the message is not properly authenticated
	if !valid {
		http.Error(w, "Invalid message signature", http.StatusUnauthorized)
		return
	}

	// Fetch host by the given path
	host, err := router.Route(r.URL.Path)
	if err != nil {
		http.Error(w, "Could not find a valid API for the request", http.StatusNotFound)
		return
	}

	// Build new path and remove prefix
	split := strings.Split(r.URL.Path, "/")
	r.URL.Path = "/" + strings.Join(split[2:], "/")

	// Call the route to serve the request
	host.handler.ServeHTTP(w, r)
}

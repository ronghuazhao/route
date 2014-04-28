// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"

	"api.umn.edu/route/router"
	"github.com/gorilla/mux"
)

func WriteJsonResponse(w http.ResponseWriter, j map[string]interface{}) http.ResponseWriter {
	w.Header().Set("Content-Type", "application/json")

	body, err := json.Marshal(j)

	if err != nil {
		logging.Log("internal", "route.error", "failed to marshal API response", "[fg-red]")
	}

	w.Write(body)

	return w
}

func NewApi(prefix string, rt *router.Router) http.Handler {
	internal := mux.NewRouter()
	api := internal.PathPrefix(prefix).Subrouter()

	RoutesHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			err := r.ParseMultipartForm(16 * 1024 * 1024)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}

			v := r.PostForm

			url, _ := url.Parse(v.Get("path"))
			proxy := httputil.NewSingleHostReverseProxy(url)

			rt.Register(v.Get("label"), v.Get("domain"), v.Get("path"), v.Get("prefix"), proxy)
		}

		body := map[string]interface{}{
			"objects": rt.Hosts,
		}

		WriteJsonResponse(w, body)

		return
	}

	RouteHandler := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		label := vars["route"]

		route := rt.Hosts[label]
		blank := &router.Host{}

		if route != *blank {
			body := map[string]interface{}{
				"objects": route,
			}

			WriteJsonResponse(w, body)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}

		return
	}

	api.HandleFunc("/routes", RoutesHandler).Methods("GET", "POST")
	api.HandleFunc("/routes/{route}", RouteHandler).Methods("GET")

	return api
}

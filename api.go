package main

import (
	"github.com/gorilla/mux"
	"github.umn.edu/umnapi/route.git/json"
	"github.umn.edu/umnapi/route.git/router"
	"net/http"
	"net/http/httputil"
	"net/url"
)

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

		response := json.NewJsonResponse(w)
		response.Write(json.Json{"objects": rt.Hosts})

		return
	}

	RouteHandler := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		label := vars["route"]

		route := rt.Hosts[label]
		blank := &router.Host{}

		if route != *blank {
			response := json.NewJsonResponse(w)
			response.Write(json.Json{"objects": route})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}

		return
	}

	api.HandleFunc("/routes", RoutesHandler).Methods("GET", "POST")
	api.HandleFunc("/routes/{route}", RouteHandler).Methods("GET")

	return api
}

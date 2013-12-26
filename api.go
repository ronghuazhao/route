package main

import (
        "github.umn.edu/umnapi/route.git/logger"
        "github.umn.edu/umnapi/route.git/router"
        "github.com/gorilla/mux"
        "net/http"
)

func NewApi(prefix string, rt *router.Router, logger *logger.Logger) http.Handler {
    internal := mux.NewRouter()
    api := internal.PathPrefix(prefix).Methods("GET").Subrouter()

    RoutesHandler := func(w http.ResponseWriter, r *http.Request) {
        response := NewJsonResponse(w)
        response.Write(Json{"objects": rt.Hosts})

        return
    }

    api.HandleFunc("/routes", RoutesHandler)

    RouteHandler := func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        label := vars["route"]

        route := rt.Hosts[label]
        blank := &router.Host{}

        if route != *blank {
            response := NewJsonResponse(w)
            response.Write(Json{"objects": route})
        } else {
            w.WriteHeader(http.StatusNotFound)
        }

        return
    }

    api.HandleFunc("/routes/{route}", RouteHandler)

    return api
}

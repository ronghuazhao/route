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
        hosts := make([]string, 0)
        for host := range rt.Hosts {
            hosts = append(hosts, host)
        }

        response := NewJsonResponse(w)
        response.Write(Json{"objects": hosts})

        return
    }

    api.HandleFunc("/routes", RoutesHandler)

    return api
}

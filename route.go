package main

import (
	"github.umn.edu/umnapi/route.git/logger"
	"github.umn.edu/umnapi/route.git/router"
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"net/http"
	"runtime"
)

// Host config file structure
type Config struct {
	Host map[string]*struct {
		Label string
	}
}

var logging *logger.Logger
var routing *router.Router
var database *sql.DB
var route_store *sqlx.DB

func init() {
    // Initiate logger
	database, _ = sql.Open("sqlite3", "/Users/ben/Code/api-auth/db/development.sqlite3")
	route_store, _ = sqlx.Connect("sqlite3", "/Users/ben/Code/api-manage/db/development.sqlite3")
    logging = logger.NewLogger("route", logger.Console)
	routing = router.NewRouter(database, route_store)
}

func main() {

	// Use all cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Create API handler
	api := NewApi("/api/v1", routing)

    // Start router
	go http.ListenAndServe(":8080", routing)
	logging.Log("internal", "route.start", "router started", "[fg-blue]")

    // Start router API
	go http.ListenAndServe(":8081", api)
	logging.Log("internal", "route.start", "api started", "[fg-blue]")

	<-make(chan int)
}

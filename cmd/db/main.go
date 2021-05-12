package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/burbokop/balanser/cmd/db/datastore"
	"github.com/burbokop/balanser/httptools"
	"github.com/burbokop/balanser/signal"
)

var port = flag.Int("port", 2361, "db port")

func main() {
	flag.Parse()

	dur := time.Minute
	db, err := datastore.NewDb("./db_segments", 1024, &dur)

	if err != nil {
		log.Fatalln("error createing db:", err)
	}

	router := httptools.NewRouter(
		[]httptools.Route{
			{
				Name:    "get-data",
				Method:  "GET",
				Pattern: "/db/{key}",
				HandlerFunc: func(writer http.ResponseWriter, request *http.Request) {
					key, err := httptools.GetStringFromPath("key", request)
					if err != nil {
						httptools.WriteError(writer, http.StatusBadRequest, err)
						return
					}
					value, err := db.Get(key)
					if err != nil {
						httptools.WriteError(writer, http.StatusInternalServerError, err)
						return
					}
					type Body struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					}
					body := Body{
						Key:   key,
						Value: value,
					}
					httptools.WriteJSONResponseOrDie(writer, http.StatusOK, body)
				},
			},
			{
				Name:    "set-data",
				Method:  "POST",
				Pattern: "/db/{key}",
				HandlerFunc: func(writer http.ResponseWriter, request *http.Request) {
					key, err := httptools.GetStringFromPath("key", request)
					if err != nil {
						httptools.WriteError(writer, http.StatusBadRequest, err)
						return
					}
					type Body struct {
						Value string `json:"value"`
					}
					body := &Body{}
					err = httptools.DecodeBodyAndClose(request.Body, body)
					if err != nil {
						httptools.WriteError(writer, http.StatusBadRequest, err)
						return
					}
					err = db.Put(key, body.Value)
					if err != nil {
						httptools.WriteError(writer, http.StatusInternalServerError, err)
						return
					}
				},
			},
		},
	)
	router.Start(*port)
	signal.WaitForTerminationSignal()
}

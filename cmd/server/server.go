package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/burbokop/balanser/cmd/server/services"
	"github.com/burbokop/balanser/httptools"
	"github.com/burbokop/balanser/signal"
)

var port = flag.Int("port", 8080, "server port")
var dbBaseAddress = flag.String("dbBaseAddress", "http://db:2361", "base address of db")

const confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
const confHealthFailure = "CONF_HEALTH_FAILURE"

func main() {
	flag.Parse()
	report := make(Report)

	dbService := services.NewDefaultDBService(*dbBaseAddress)

	router := httptools.NewRouter(
		[]httptools.Route{
			{
				Name:    "health",
				Method:  "GET",
				Pattern: "/health",
				HandlerFunc: func(rw http.ResponseWriter, r *http.Request) {
					rw.Header().Set("content-type", "text/plain")
					if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
						rw.WriteHeader(http.StatusInternalServerError)
						_, _ = rw.Write([]byte("FAILURE"))
					} else {
						rw.WriteHeader(http.StatusOK)
						_, _ = rw.Write([]byte("OK"))
					}
				},
			},
			{
				Name:    "get-some-data",
				Method:  "GET",
				Pattern: "/api/v1/some-data/{id:[0-9]+}",
				HandlerFunc: func(writer http.ResponseWriter, request *http.Request) {
					key, err := httptools.GetStringFromQuery("key", true, request)
					if err != nil {
						httptools.WriteError(writer, http.StatusBadRequest, err)
						return
					}

					respDelayString := os.Getenv(confResponseDelaySec)
					if delaySec, parseErr := strconv.Atoi(respDelayString); parseErr == nil && delaySec > 0 && delaySec < 300 {
						time.Sleep(time.Duration(delaySec) * time.Second)
					}

					report.Process(request)

					result, err := dbService.GetValue(key)
					if err != nil {
						httptools.WriteError(writer, http.StatusInternalServerError, err)
						return
					}
					httptools.WriteJSONResponseOrDie(writer, http.StatusOK, result)
				},
			},
			{
				Name:    "set-some-data",
				Method:  "POST",
				Pattern: "/api/v1/some-data",
				HandlerFunc: func(writer http.ResponseWriter, request *http.Request) {
					key, err := httptools.GetStringFromQuery("key", true, request)
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

					result, err := dbService.SetValue(key, body.Value)
					if err != nil {
						httptools.WriteError(writer, http.StatusInternalServerError, err)
						return
					}
					httptools.WriteJSONResponseOrDie(writer, http.StatusOK, result)
				},
			},
		},
	)
	router.Start(*port)
	signal.WaitForTerminationSignal()
}

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/burbokop/balanser/signal"
)

var port = flag.Int("port", 8080, "server port")

const confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
const confHealthFailure = "CONF_HEALTH_FAILURE"

func main() {
	report := make(Report)
	router := NewRouter(
		[]Route{
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
				Name:    "some-data",
				Method:  "GET",
				Pattern: "/api/v1/some-data/{id:[0-9]+}",
				HandlerFunc: func(rw http.ResponseWriter, r *http.Request) {
					log.Printf("request recieved (url: %s)", r.URL.Path)
					respDelayString := os.Getenv(confResponseDelaySec)
					if delaySec, parseErr := strconv.Atoi(respDelayString); parseErr == nil && delaySec > 0 && delaySec < 300 {
						time.Sleep(time.Duration(delaySec) * time.Second)
					}

					report.Process(r)

					rw.Header().Set("content-type", "application/json")
					rw.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(rw).Encode([]string{"1", "2"})
				},
			},
		},
	)
	router.Start(*port)
	signal.WaitForTerminationSignal()
}

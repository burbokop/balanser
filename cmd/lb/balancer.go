package main

import (
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/burbokop/balanser/httptools"
	"github.com/burbokop/balanser/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

type Server struct {
	Name    string
	IsAlive bool
}

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []Server{
		{Name: "server1:8080", IsAlive: false},
		{Name: "server2:8080", IsAlive: false},
		{Name: "server3:8080", IsAlive: false},
	}
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func chooseServer(serversPool []Server, url *url.URL) (*uint64, error) {
	index := hash64(url.Path) % uint64(len(serversPool))
	for i := 0; i < len(serversPool) && !serversPool[index].IsAlive; i++ {
		index = (index + 1) % uint64(len(serversPool))
	}
	if serversPool[index].IsAlive {
		return &index, nil
	} else {
		return nil, fmt.Errorf("balancer: no alive servers found")
	}
}

func main() {
	flag.Parse()

	for i := range serversPool {
		go func(i int) {
			for range time.Tick(10 * time.Second) {
				serversPool[i].IsAlive = health(serversPool[i].Name)
			}
		}(i)
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		index, err := chooseServer(serversPool, r.URL)
		if err != nil {
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Write([]byte(err.Error()))
			return
		}
		forward(serversPool[*index].Name, rw, r)
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}

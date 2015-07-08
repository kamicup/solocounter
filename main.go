package main

import (
	"flag"
	"fmt"
	"github.com/kamicup/solocounter/server"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"
)

/**
 * A micro-service for concurrent access counter.
 *
 * Cross-compile:
 * $ GOOS=linux   GOARCH=amd64 go build -o solocounter.linux.amd64
 * $ GOOS=darwin  GOARCH=amd64 go build -o solocounter.darwin.amd64
 * $ GOOS=windows GOARCH=amd64 go build -o solocounter.windows.amd64
 *
 * Usage:
 * $ PORT=80 solocounter -time-window=5 -interval=10
 *
 * Check service status:
 * $ curl http://localhost:80/_stats/
 *
 * Run GC:
 * $ curl http://localhost:80/_gc/
 *
 * Check heap profile with go-tool:
 * $ go tool pprof --text http://localhost:80/debug/pprof/heap
 *
 */
func main() {
	var (
		window     int
		interval   int
		parallel   bool
		simulate   bool
		redis      string
		redis_auth string
		redis_node string
		verbose    bool
	)
	flag.IntVar(&window, "time-window", 1, "Time window [min]")
	flag.IntVar(&interval, "interval", 1, "Cleanup interval [sec]")
	flag.BoolVar(&parallel, "parallel", false, "Parallel cleanup mode")
	flag.BoolVar(&simulate, "simulate", false, "Simulate 1000 [sites] x 10 [req/sec]")
	flag.StringVar(&redis, "redis", "", "Redis server (ex.) :6379")
	flag.StringVar(&redis_auth, "redis-auth", "", "Redis password")
	flag.StringVar(&redis_node, "redis-node", "", "Redis node-name")
	flag.BoolVar(&verbose, "v", false, "Show verbose log")
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT not set")
	}

	storage := server.NewStorage(time.Second*time.Duration(interval), time.Minute*time.Duration(window))
	storage.Verbose = verbose
	storage.Clean(parallel)

	if simulate {
		terminator := storage.Simulate(1000, 10) // Simulate 1000 [sites] x 10 [req/sec]
		http.HandleFunc("/_terminate", func(w http.ResponseWriter, r *http.Request) {
			terminator()
		})
	}

	if redis != "" {
		storage.PubSub(redis, redis_auth, redis_node)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var address string
		xff, ok := r.Header["X-Forwarded-For"]
		if ok && len(xff) > 0 {
			address = xff[0]
		} else {
			address = r.RemoteAddr
		}
		num := storage.Push(r.URL.Path, address)

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%d", num)
	})
	http.Handle("/_stats/", storage)
	http.Handle("/_debug/", http.RedirectHandler("/debug/pprof/", http.StatusFound))
	http.HandleFunc("/_gc", func(w http.ResponseWriter, r *http.Request) {
		runtime.GC()
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

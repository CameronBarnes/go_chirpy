package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type stats struct {
	fileserverHits atomic.Int32
}

func (stats *stats) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		stats.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (stats *stats) hitsMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	fmt.Fprintf(w, "Hits: %v", stats.fileserverHits.Load())
}

func (stats *stats) resetMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	stats.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func healthcheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	stats := stats{}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", stats.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.HandleFunc("/metrics", stats.hitsMetricsHandler)
	mux.HandleFunc("/healthz", healthcheck)
	mux.HandleFunc("/reset", stats.resetMetricsHandler)
	server := http.Server{Handler: mux, Addr: ":8080"}
	server.ListenAndServe()
}

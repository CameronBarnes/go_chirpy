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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (stats *stats) hitsMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	fmt.Fprintf(w, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, stats.fileserverHits.Load())
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
	mux.HandleFunc("GET /admin/metrics", stats.hitsMetricsHandler)
	mux.HandleFunc("POST /admin/reset", stats.resetMetricsHandler)
	mux.HandleFunc("GET /api/healthz", healthcheck)
	server := http.Server{Handler: mux, Addr: ":8080"}
	server.ListenAndServe()
}

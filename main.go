package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func validateChirp(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Body string `json:"body"`
	}
	type err struct {
		Error string `json:"error"`
	}
	type ok struct {
		Valid bool `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	args := params{}
	if error := decoder.Decode(&args); error != nil {
		log.Printf("Error decoding parameters: %s", error)
		w.WriteHeader(500)
		dat, error2 := json.Marshal(err{Error: error.Error()})
		if error2 != nil {
			log.Printf("Error encoding output error: %s for error %s", error2, error)
			return
		}
		w.Write(dat)
		return
	}

	if len(args.Body) > 140 {
		w.WriteHeader(400)
		dat, error2 := json.Marshal(err{Error: "Chirp is too long"})
		if error2 != nil {
			log.Printf("Error encoding 'Chirp is too long' response with err: %s", error2)
			return
		}
		w.Write(dat)
		return
	}

	w.WriteHeader(200)
	dat, error2 := json.Marshal(ok{Valid: true})
	if error2 != nil {
		log.Printf("Error encoding success response with err: %s", error2)
		return
	}
	w.Write(dat)

}

func main() {
	stats := stats{}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", stats.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)
	mux.HandleFunc("GET /admin/metrics", stats.hitsMetricsHandler)
	mux.HandleFunc("POST /admin/reset", stats.resetMetricsHandler)
	mux.HandleFunc("GET /api/healthz", healthcheck)
	server := http.Server{Handler: mux, Addr: ":8080"}
	server.ListenAndServe()
}

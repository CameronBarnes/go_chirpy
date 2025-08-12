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

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type err struct {
		Error string `json:"error"`
	}
	w.WriteHeader(code)
	dat, error2 := json.Marshal(err{Error: msg})
	if error2 != nil {
		log.Printf("Error encoding response error: %s for error %s", error2, msg)
		return
	}
	w.Write(dat)
}

func respondWithJSON[T any](w http.ResponseWriter, code int, payload T) {
	w.WriteHeader(code)
	dat, error2 := json.Marshal(payload)
	if error2 != nil {
		log.Printf("Error encoding success response with err: %s", error2)
		return
	}
	w.Write(dat)
}

func validateChirp(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Body string `json:"body"`
	}

	type ok struct {
		Valid bool `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	args := params{}
	if error := decoder.Decode(&args); error != nil {
		log.Printf("Error decoding parameters: %s", error)
		respondWithError(w, 500, fmt.Sprintf("Error decoding parameters: %s", error))
		return
	}

	if len(args.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	respondWithJSON[ok](w, 200, ok{Valid: true})

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

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/cameronbarnes/go_chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
}

func (c *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (c *apiConfig) hitsMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	fmt.Fprintf(w, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, c.fileserverHits.Load())
}

func (c *apiConfig) resetHandler(w http.ResponseWriter, _ *http.Request) {
	c.fileserverHits.Store(0)
	c.db.DeleteAll(context.Background())
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (c *apiConfig) addUser(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Email string `json:"email"`
	}
	arg, err := handleParse[req](w, r)
	if err != nil {
		return
	}
	user, err := c.db.CreateUser(context.Background(), arg.Email)
	if err != nil {
		log.Printf("Failed to create user with error: %s", err.Error())
		respondWithError(w, 500, "Failed to create user")
		return
	}
	respondWithJSON(w, 201, user)
}

func healthcheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func handleParse[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	decoder := json.NewDecoder(r.Body)
	var val T
	if err := decoder.Decode(&val); err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(w, 500, fmt.Sprintf("Error decoding body: %s", err))
		return val, err
	}
	return val, nil
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

func cleanStr(input string, bad string) string {
	out := []string{}
	for str := range strings.SplitSeq(input, " ") {
		if strings.ToLower(str) == bad {
			out = append(out, "****")
		} else {
			out = append(out, str)
		}
	}
	return strings.Join(out, " ")
}

func validateChirp(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Body string `json:"body"`
	}

	type ok struct {
		Cleaned_Body string `json:"cleaned_body"`
	}

	args, err := handleParse[params](w, r)
	if err != nil {
		return
	}

	if len(args.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	respondWithJSON(w, 200, ok{Cleaned_Body: cleanStr(cleanStr(cleanStr(args.Body, "kerfuffle"), "sharbert"), "fornax")})

}

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cfg := apiConfig{db: database.New(db)}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)
	mux.HandleFunc("POST /api/users", cfg.addUser)
	mux.HandleFunc("GET /admin/metrics", cfg.hitsMetricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.resetHandler)
	mux.HandleFunc("GET /api/healthz", healthcheck)
	server := http.Server{Handler: mux, Addr: ":8080"}
	server.ListenAndServe()
}

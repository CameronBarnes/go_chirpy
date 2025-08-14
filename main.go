package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cameronbarnes/go_chirpy/internal/auth"
	"github.com/cameronbarnes/go_chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	jwtSecret      string
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
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	arg, err := handleParse[req](w, r)
	if err != nil {
		return
	}
	hash, err := auth.HashPassword(arg.Password)
	if err != nil {
		respondWithError(w, 400, "Password is not valid")
		return
	}
	user, err := c.db.CreateUser(context.Background(), database.CreateUserParams{Email: strings.ToLower(arg.Email), HashedPassword: hash})
	if err != nil {
		log.Printf("Failed to create user with error: %s", err.Error())
		respondWithError(w, 500, "Failed to create user")
		return
	}
	respondWithJSON(w, 201, user)
}

func (c *apiConfig) login(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	arg, err := handleParse[req](w, r)
	if err != nil {
		return
	}
	user, err := c.db.GetUserFromEmail(context.Background(), strings.ToLower(arg.Email))
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	check := auth.CheckPassword(arg.Password, user.HashedPassword)
	if check != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	token, err := auth.MakeJWT(user.ID, c.jwtSecret, time.Hour)
	if err != nil {
		log.Printf("Failed to make JWT with error :%v", err)
		respondWithError(w, 500, "Failed to build auth")
		return
	}
	refresh_str, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Failed to make refresh token string with error: %v", err)
		respondWithError(w, 500, "Failed to build auth")
		return
	}
	_, err2 := c.db.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{UserID: user.ID, Token: refresh_str, ExpiresAt: time.Now().Add(time.Hour * 24 * 60)})
	if err2 != nil {
		log.Printf("Failed to create refresh token with error: %v", err2)
		respondWithError(w, 500, "Failed to build auth")
		return
	}
	userOutFromLogin(w, 200, user, token, refresh_str)
}

func (c *apiConfig) addChirp(w http.ResponseWriter, r *http.Request, user database.GetUserRow) {
	type chirpArgs struct {
		Body string `json:"body"`
	}
	arg, err := handleParse[chirpArgs](w, r)
	if err != nil {
		respondWithError(w, 400, "Invalid Request")
		return
	}
	body, err := validateChirp(arg.Body)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	chirp, err := c.db.AddChirp(context.Background(), database.AddChirpParams{Body: body, UserID: user.ID})
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	respondWithJSON(w, 201, chirp)
}

func (c *apiConfig) updateUser(w http.ResponseWriter, r *http.Request, user database.GetUserRow) {
	type updateArgs struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	args, err := handleParse[updateArgs](w, r)
	if err != nil {
		respondWithError(w, 400, "Invalid Request")
		return
	}
	hashed, err := auth.HashPassword(args.Password)
	if err != nil {
		respondWithError(w, 400, "Password is not valid")
		return
	}
	updated, err := c.db.UpdateUser(context.Background(), database.UpdateUserParams{ID: user.ID, Email: args.Email, HashedPassword: hashed})
	respondWithJSON(w, 200, updated)
}

func (c *apiConfig) getChirps(w http.ResponseWriter, _ *http.Request) {
	chirps, err := c.db.AllChirps(context.Background())
	if err != nil {
		log.Printf("Failed to get chirps with err: %s", err)
		respondWithError(w, 500, "Failed to get chirps")
		return
	}
	respondWithJSON(w, 200, chirps)
}

func (c *apiConfig) getChirp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpID")
	if id == "" {
		respondWithError(w, 404, "Chirp Not Found")
		return
	}
	uuid, err := uuid.Parse(id)
	if err != nil {
		respondWithError(w, 400, "UUID provided is not valid")
		return
	}
	chirp, err := c.db.GetChirp(context.Background(), uuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "Chirp Not Found")
		} else {
			log.Println(err)
			respondWithError(w, 500, "Failed to get Chirp")
		}
		return
	}
	respondWithJSON(w, 200, chirp)
}

func userOutFromLogin(w http.ResponseWriter, code int, obj database.User, jwt, refresh string) {
	type user_out struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}
	respondWithJSON(w, code, user_out{ID: obj.ID, CreatedAt: obj.CreatedAt, UpdatedAt: obj.UpdatedAt, Email: obj.Email, Token: jwt, RefreshToken: refresh})
}

func (c *apiConfig) refresh(w http.ResponseWriter, r *http.Request) {
	type out_struct struct {
		Token string `json:"token"`
	}
	token_str, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	refresh_token, err := c.db.GetRefreshToken(context.Background(), token_str)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	if refresh_token.RevokedAt.Valid || refresh_token.ExpiresAt.Before(time.Now()) {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	token, err := auth.MakeJWT(refresh_token.UserID, c.jwtSecret, time.Hour)
	if err != nil {
		log.Printf("Failed to make JWT with error :%v", err)
		respondWithError(w, 500, "Failed to build auth")
		return
	}
	respondWithJSON(w, 200, out_struct{Token: token})
}

func (c *apiConfig) revoke(w http.ResponseWriter, r *http.Request) {
	token_str, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	err2 := c.db.ExpireToken(context.Background(), token_str)
	if err2 != nil {
		log.Printf("Failed to revoke token with error: %v", err2)
		respondWithError(w, 500, "Failed to revoke token")
		return
	}
	w.WriteHeader(204)
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

func validateChirp(text string) (string, error) {
	if len(text) > 140 {
		return "", errors.New("Chirp is too long")
	}

	return cleanStr(cleanStr(cleanStr(text, "kerfuffle"), "sharbert"), "fornax"), nil
}

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cfg := apiConfig{db: database.New(db), jwtSecret: jwtSecret}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.HandleFunc("POST /api/chirps", cfg.getUserMiddleware(cfg.addChirp))
	mux.HandleFunc("GET /api/chirps", cfg.getChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.getChirp)
	mux.HandleFunc("POST /api/users", cfg.addUser)
	mux.HandleFunc("PUT /api/users", cfg.getUserMiddleware(cfg.updateUser))
	mux.HandleFunc("POST /api/login", cfg.login)
	mux.HandleFunc("POST /api/refresh", cfg.refresh)
	mux.HandleFunc("POST /api/revoke", cfg.revoke)
	mux.HandleFunc("GET /admin/metrics", cfg.hitsMetricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.resetHandler)
	mux.HandleFunc("GET /api/healthz", healthcheck)
	server := http.Server{Handler: mux, Addr: ":8080"}
	server.ListenAndServe()
}

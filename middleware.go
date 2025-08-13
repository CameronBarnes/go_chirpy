package main

import (
	"context"
	"log"
	"net/http"

	"github.com/cameronbarnes/go_chirpy/internal/auth"
	"github.com/cameronbarnes/go_chirpy/internal/database"
)

func (c *apiConfig) getUserMiddleware(next func(w http.ResponseWriter, r *http.Request, user database.GetUserRow)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token_str, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, 401, "Unauthorized")
			return
		}
		id, err := auth.ValidateJWT(token_str, c.jwtSecret)
		if err != nil {
			respondWithError(w, 401, "Unauthorized")
			return
		}
		user, err := c.db.GetUser(context.Background(), id)
		if err != nil {
			log.Printf("Failed to get user with error: %v", err)
			respondWithError(w, 401, "Unauthorized")
			return
		}
		next(w, r, user)
	}
}

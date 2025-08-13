package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	val := headers.Get("Authorization")
	if val == "" {
		return "", errors.New("Auth header is missing")
	}
	return strings.TrimPrefix(val, "Bearer "), nil
}

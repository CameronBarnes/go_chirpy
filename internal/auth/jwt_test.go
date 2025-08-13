package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTValid(t *testing.T) {
	secret := "I like cats456789"
	id := uuid.New()
	jwt, err := MakeJWT(id, secret, time.Minute * 5)
	if err != nil {
		t.Error(err)
	}
	id_out, err := ValidateJWT(jwt, secret)
	if err != nil {
		t.Error(err)
	}
	if id_out != id {
		t.Errorf("Output UUID from ValidateJWT does not match input to MakeJWT")
	}
}

func TestJWTExpires(t *testing.T) {
	secret := "I like cats456789"
	id := uuid.New()
	jwt, err := MakeJWT(id, secret, time.Minute * -5)
	if err != nil {
		t.Error(err)
	}
	_, err2 := ValidateJWT(jwt, secret)
	if err2 == nil {
		t.Errorf("JWT should fail to validate due to expiration")
	}
}

func TestJWTSecretMatters(t *testing.T) {
	secret := "I like cats456789"
	id := uuid.New()
	jwt, err := MakeJWT(id, secret, time.Minute * -5)
	if err != nil {
		t.Error(err)
	}
	_, err2 := ValidateJWT(jwt, "dogs are best1234")
	if err2 == nil {
		t.Errorf("JWT should fail to validate due to bad secret")
	}
}

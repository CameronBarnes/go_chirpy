package auth

import (
	"testing"
)

func TestHashEqual(t *testing.T) {
	pass := "catis GOOD"
	h, err := HashPassword(pass)
	if err != nil {
		t.Error(err)
	}
	err2 := CheckPassword(pass, h)
	if err2 != nil {
		t.Error(err2)
	}
}

func TestHashNotEqual(t *testing.T) {
	passA := "Cat is Best!"
	h, err := HashPassword(passA)
	if err != nil {
		t.Error(err)
	}
	passB := "Dog is Best!"
	err2 := CheckPassword(passB, h)
	if err2 == nil {
		t.Errorf("Password matched hash it shouldnt have")
	}
}

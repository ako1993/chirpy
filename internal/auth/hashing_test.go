package auth

import (
	"fmt"
	"testing"
)

func TestHashing(t *testing.T) {
	plain_text_pw := "password"
	hashed_pw, err := HashPassword(plain_text_pw)
	if err != nil {
		fmt.Println(err)
	}
	is_valid, err := CheckPasswordHash("password", hashed_pw)
	if err != nil {
		fmt.Println(err)
	}
	expected := true
	if is_valid != expected {
		t.Errorf("Result: %v; Expected: %v", is_valid, expected)
	}
}

func TestHashing2(t *testing.T) {
	plain_text_pw := "kjdsbjckedbkjfbwkjsbsknlcsdknldcnklanklnka3y268278"
	hashed_pw, err := HashPassword(plain_text_pw)
	if err != nil {
		fmt.Println(err)
	}
	is_valid, err := CheckPasswordHash("kjdsbjckedbkjfbwkjsbsknlcsdknldcnklanklnka3y268278", hashed_pw)
	if err != nil {
		fmt.Println(err)
	}
	expected := true
	if is_valid != expected {
		t.Errorf("Result: %v; Expected: %v", is_valid, expected)
	}
}

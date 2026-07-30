package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTCreation(t *testing.T) {
	id := uuid.New()
	tokensecret := "secret"
	expiresIn := 2 * time.Minute
	jwt, err := MakeJWT(id, tokensecret, expiresIn)
	if err != nil {
		fmt.Println(err)
	}
	user_id, err := ValidateJWT(jwt, tokensecret)
	if err != nil {
		fmt.Println(err)
	}
	if user_id != id {
		t.Errorf("Initial id: %v; Verified id: %v", id, user_id)
	}

}

func TestInvalidKey(t *testing.T) {
	id := uuid.New()
	tokensecret := "secret1"
	expiresIn := 2 * time.Minute
	jwt, err := MakeJWT(id, tokensecret, expiresIn)
	if err != nil {
		fmt.Println(err)
	}
	new_id, err := ValidateJWT(jwt, "notmysecret")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(new_id, err)
}

func TestExpiredToken(t *testing.T) {
	id := uuid.New()
	tokensecret := "secret1"
	expiresIn := -2 * time.Hour
	jwt, err := MakeJWT(id, tokensecret, expiresIn)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(jwt, err)
}

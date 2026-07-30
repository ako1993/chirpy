package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	key := []byte(tokenSecret)
	new_token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	})
	expiration_time, err := new_token.Claims.GetExpirationTime()
	if expiration_time.Unix() < time.Now().Unix() {
		return "", errors.New("Token is expired")
	}
	signed_JWT, err := new_token.SignedString(key)
	if err != nil {
		return "", err
	}
	return signed_JWT, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	key := []byte(tokenSecret)
	my_claims := &jwt.RegisteredClaims{}
	parsed_token, err := jwt.ParseWithClaims(tokenString, my_claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	if !parsed_token.Valid {
		return uuid.Nil, errors.New("Invalid token")
	}
	subject, err := my_claims.GetSubject()
	parsed_subject, err := uuid.Parse(subject)

	return parsed_subject, nil

}

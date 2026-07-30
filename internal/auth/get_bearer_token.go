package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	key := http.CanonicalHeaderKey("Authorization")
	if value, ok := headers[key]; ok {
		new_value := strings.Join(value, " ")
		new_value = strings.TrimPrefix(new_value, "Bearer")
		new_value = strings.TrimSpace(new_value)
		return new_value, nil
	} else {
		return "", errors.New("Authorization Token not found in header")
	}
}

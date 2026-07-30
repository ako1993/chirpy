package auth

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGetBearerToken(t *testing.T) {
	url := "http://localhost:8080/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer your_token_here")
	req.Header.Set("X-Custom-Header", "MyValue")
	auth_value, err := GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
	}
	if auth_value != "your_token_here" {
		t.Errorf("auth_value: %v; Expected Value: %v", auth_value, "your_token_here")
	}
}

func TestAuthorizationHeaderNotPresent(t *testing.T) {
	url := "http://localhost:8080/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "MyValue")
	auth_value, err := GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(auth_value)
	fmt.Println(err)
}

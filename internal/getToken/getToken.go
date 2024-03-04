package gettoken

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenResponse struct {
	Token string `json:"token"`
}

func GetToken(w http.ResponseWriter, r *http.Request) {
	// Create a new token with claims
	claims := jwt.MapClaims{
		// "admin": true,
		"exp": time.Now().Add(time.Hour * 24).Unix(),

		// "name": "Ilia",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//надо наверно запихнуть в .env, но делать я конечно же этого не буду
	secretKey := []byte("avadakedavra")
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := TokenResponse{Token: tokenString}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

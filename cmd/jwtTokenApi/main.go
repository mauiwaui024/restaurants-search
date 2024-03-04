package main

import (
	"elasticsearch/internal/db"
	gettoken "elasticsearch/internal/getToken"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type DataForJSON struct {
	Name   string     `json:"name"`
	Places []db.Place `json:"places"`
}

func main() {
	http.HandleFunc("/api/recommend", requireToken(fetchPlacesWithSorting))
	http.HandleFunc("/api/get_token", gettoken.GetToken)
	log.Fatal(http.ListenAndServe(":8888", nil))
}

func requireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the token from the Authorization header
		tokenString := r.Header.Get("Authorization")
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		if tokenString == "" {
			http.Error(w, "Unauthorized1, error 401", http.StatusUnauthorized)
			return
		}

		// Verify the token
		secretKey := []byte("avadakedavra")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})
		// fmt.Println("Token issssssssssssss", token, err)

		fmt.Println("Token Claims:", token.Claims)
		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized2, error 401", http.StatusUnauthorized)
			return
		}

		// Call the next handler if the token is valid
		next.ServeHTTP(w, r)
	}
}

func fetchPlacesWithSorting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	es, err1 := db.NewElasticStore()
	if err1 != nil {
		log.Fatal(err1)
	}
	// fmt.Println(es)
	var err error
	var response DataForJSON
	// забираем параметры lat и lon из запроса
	latParam := r.URL.Query().Get("lat")
	lonParam := r.URL.Query().Get("lon")

	// проверяем, что параметры lat и lon не пусты
	if latParam == "" || lonParam == "" {
		http.Error(w, `{"error": "lat and lon parameters are required"}`, http.StatusBadRequest)
		return
	}

	// преобразуем параметры lat и lon в числовой формат
	lat, err := strconv.ParseFloat(latParam, 64)
	if err != nil {
		http.Error(w, `{"error": "Invalid 'lat' value"}`, http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonParam, 64)
	if err != nil {
		http.Error(w, `{"error": "Invalid 'lon' value"}`, http.StatusBadRequest)
		return
	}
	fmt.Println(lon, lat)

	//вызываем функцию GetPlaces
	response.Places, err = es.GetPlacesWithSorting(lon, lat)

	// fmt.Println(response.Page)
	response.Name = "recommend"
	//надо как-то запихнуть этот стракт в json
	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	//отправляем ответ на клиент
	w.Write(jsonData)
}

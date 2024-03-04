package main

import (
	"elasticsearch/internal/db"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
)

type DataForJSON struct {
	Name     string     `json:"name"`
	Total    int        `json:"total"`
	Places   []db.Place `json:"places"`
	PrevPage int        `json:"prev_page"`
	NextPage int        `json:"next_page"`
	LastPage int        `json:"last_page"`
	Page     int        `json:"page"`
}

func main() {
	http.HandleFunc("/api/places", fetchPlacesJSON)
	log.Fatal(http.ListenAndServe(":8888", nil))
}

func fetchPlacesJSON(w http.ResponseWriter, r *http.Request) {
	es, err1 := db.NewElasticStore()
	if err1 != nil {
		log.Fatal(err1)
	}

	var err error
	var response DataForJSON
	// забираем квери параметр
	if response.Page, err = strconv.Atoi(r.URL.Query().Get("page")); err != nil {
		// здесь же шлем нахрен если не число подали
		http.Error(w, `{"error": "Invalid 'page' value"}`, http.StatusBadRequest)
		return
	}

	limit := 10
	offset := (response.Page - 1) * limit
	//вызываем функцию GetPlaces
	response.Places, response.Total, err = es.GetPlaces(limit, offset)
	response.LastPage = int(math.Ceil(float64(response.Total) / float64(limit)))
	// проверяем корректность значения 'page'
	if response.Page < 0 || response.Page > response.LastPage {
		http.Error(w, `{"error": "Invalid 'page' value"}`, http.StatusBadRequest)
		return
	}
	response.Name = "places"
	//надо как-то запихнуть этот стракт в json
	response.PrevPage = response.Page - 1
	response.NextPage = response.Page + 1

	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Set Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")
	//отправляем ответ на клиент
	w.Write(jsonData)
}

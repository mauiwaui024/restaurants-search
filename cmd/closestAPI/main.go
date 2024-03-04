package main

import (
	"elasticsearch/internal/db"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type DataForJSON struct {
	Name   string     `json:"name"`
	Places []db.Place `json:"places"`
}

func main() {
	http.HandleFunc("/api/recommend", fetchPlacesWithSorting)
	log.Fatal(http.ListenAndServe(":8888", nil))
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

	//вызываем функцию GetPlaces
	response.Places, err = es.GetPlacesWithSorting(lon, lat)

	// fmt.Println(response.Page)
	response.Name = "recommend"
	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	//отправляем ответ на клиент
	w.Write(jsonData)
}

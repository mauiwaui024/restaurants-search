package main

import (
	"elasticsearch/internal/db"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
)

type DataForHTML struct {
	Places []db.Place
	Total  int
	Page   int
	Last   int
}

func main() {
	http.HandleFunc("/", fetchPlaces)
	log.Fatal(http.ListenAndServe(":8888", nil))
}

func fetchPlaces(w http.ResponseWriter, r *http.Request) {
	es, err1 := db.NewElasticStore()
	if err1 != nil {
		log.Fatal(err1)
	}

	var err error
	var response DataForHTML
	//забираем квери параметр
	if response.Page, err = strconv.Atoi(r.URL.Query().Get("page")); err != nil {
		//здесь же шлем нахрен если не число подали
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	limit := 10
	offset := (response.Page - 1) * limit
	//вызываем функцию GetPlaces
	response.Places, response.Total, err = es.GetPlaces(limit, offset)
	//парсим нашу страницу
	response.Last = int(math.Ceil(float64(response.Total) / float64(limit)))
	tmpl, err := template.New("index.html").Funcs(
		template.FuncMap{
			"sum": sum,
			"sub": sub,
		},
	).ParseFiles("internal/htmlUI/index.html")

	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error1", http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, response)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error2", http.StatusInternalServerError)
		return
	}

}

func sum(x, y int) int {
	return x + y
}

func sub(x, y int) int {
	return x - y
}

package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch"
)

type Place struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Phone    string   `json:"phone"`
	Location Geopoint `json:"location"`
}
type Geopoint struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

type ElasticStore struct {
	Es *elasticsearch.Client
}

func NewElasticStore() (*ElasticStore, error) {
	//дефолтный
	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		return nil, err
	}
	return &ElasticStore{es}, nil
}

// интерфейс
type Store interface {
	// returns a list of items, a total number of hits and (or) an error in case of one
	GetPlaces(limit int, offset int) ([]Place, int, error)
	GetPlacesWithSorting(limit int, lon float64, lat float64) ([]Place, error)
}

func (store *ElasticStore) GetPlaces(limit int, offset int) ([]Place, int, error) {
	query := map[string]interface{}{
		"size": limit,
		"from": offset,
	}
	///теперь мы мапу эту конвертим в json
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, 0, err
	}

	// fmt.Println("HELLLOOOO", string(queryJSON))

	res, err := store.Es.Search(
		store.Es.Search.WithIndex("places"), // Replace with your actual index name
		store.Es.Search.WithBody(strings.NewReader(string(queryJSON))),
		store.Es.Search.WithSize(limit),
		store.Es.Search.WithFrom(offset),
		store.Es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, err
	}

	defer res.Body.Close()
	if res.IsError() {
		return nil, 0, fmt.Errorf("Elasticsearch error: %s", res.String())
	}
	//теперь надо раскодировать res который пришел в json и запихнуть это все
	var ResBodyResult map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&ResBodyResult); err != nil {
		return nil, 0, err
	}

	totalHits := int(ResBodyResult["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))

	// fmt.Println(ResBodyResult["hits"])
	//просто очень странное .dot notation, навигируем до
	hits := ResBodyResult["hits"].(map[string]interface{})["hits"].([]interface{})
	//создаем slice с местами
	places := make([]Place, 0, len(hits))

	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"]
		placeBytes, err := json.Marshal(source)
		if err != nil {
			continue
		}

		var place Place
		if err := json.Unmarshal(placeBytes, &place); err != nil {
			continue
		}
		places = append(places, place)
	}

	return places, totalHits, nil
}

func (store *ElasticStore) GetPlacesWithSorting(lon float64, lat float64) ([]Place, error) {
	sortConfig := []map[string]interface{}{
		{
			"_geo_distance": map[string]interface{}{
				"location": map[string]interface{}{
					"lat": lat,
					"lon": lon,
				},
				"order":           "asc",
				"unit":            "km",
				"mode":            "min",
				"distance_type":   "arc",
				"ignore_unmapped": true,
			},
		},
	}

	// Construct the search query
	query := map[string]interface{}{
		"size": 3,
		"sort": sortConfig,
	}

	// Convert the query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	// Perform the search with sorting
	res, err := store.Es.Search(
		store.Es.Search.WithIndex("places"),
		store.Es.Search.WithBody(strings.NewReader(string(queryJSON))),
	)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	// роспакоука нахуй
	var ResBodyResult map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&ResBodyResult); err != nil {
		return nil, err
	}
	// вытаскивывем совпадения
	hits := ResBodyResult["hits"].(map[string]interface{})["hits"].([]interface{})

	places := make([]Place, 0)

	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"]
		placeBytes, err := json.Marshal(source)
		if err != nil {
			continue
		}

		var place Place
		if err := json.Unmarshal(placeBytes, &place); err != nil {
			continue
		}

		places = append(places, place)
	}

	return places, nil
}

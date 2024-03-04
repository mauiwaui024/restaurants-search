package dataloader

import (
	"bytes"
	"context"
	"elasticsearch/internal/db"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dustin/go-humanize"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

func InitElasticClient() (*elasticsearch.Client, error) {
	//крч создаем сущность(в виде гномика бл) elasticsearch клиента
	//нахрен отключили все сертификаты, чтобы отправлять запросы на локальный сервак без авторизации
	retryBackoff := backoff.NewExponentialBackOff()
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
		// Retry on 429 TooManyRequests statuses
		RetryOnStatus: []int{502, 503, 504, 429},
		// Configure the backoff function
		//
		RetryBackoff: func(i int) time.Duration {
			if i == 1 {
				retryBackoff.Reset()
			}
			return retryBackoff.NextBackOff()
		},
		// Retry up to 5 attempts
		//
		MaxRetries: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Elasticsearch client: %s", err)
	}
	return es, nil

}

func CreateIndex(es *elasticsearch.Client) error {
	IndexName := "places"
	res, err := es.Indices.Create(IndexName)
	if err != nil {
		return fmt.Errorf("%s", res)
	}
	defer res.Body.Close()
	if res.IsError() {
		//походу ошибку из функции лучше всего возвращать с fmt.Errorf()
		return fmt.Errorf("Error creating index: %s", res)
	}
	return nil
}

func AddMapping(es *elasticsearch.Client) error {
	mapping := `{
		"properties": {
			"name": {
				"type": "text"
			},
			"address": {
				"type": "text"
			},
			"phone": {
				"type": "text"
			},
			"location": {
				"type": "geo_point"
			}
		}
	}`
	putMappingRes, err := es.Indices.PutMapping(
		[]string{"places"},
		strings.NewReader(mapping),
		es.Indices.PutMapping.WithPretty(),
	)
	if err != nil {
		return fmt.Errorf("Error creating mapping1: %s", err)
	}
	defer putMappingRes.Body.Close()

	// Check if the request was successful
	if putMappingRes.IsError() {
		return fmt.Errorf("Error creating mapping2: %s", putMappingRes)
	}
	return nil
}

func ReadCsv() ([]db.Place, error) {
	var PlacesSlice []db.Place
	// fmt.Println(PlacesSlice)
	//
	file, err := os.Open("../materials/data.csv")
	if err != nil {
		return nil, fmt.Errorf("Error opening: %s", err)
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	csvReader.Comma = '\t'
	records, err := csvReader.ReadAll()
	// var id int = 0
	firstRow := true

	for _, record := range records {
		if firstRow {
			firstRow = false
			continue
		}
		lon, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			fmt.Println("lon is  ", record[4])
			return nil, fmt.Errorf("invalid Longitude: %s", record[4])
		}
		lat, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid Latitude: %s", record[5])
		}
		geo := db.Geopoint{
			Lon: lon,
			Lat: lat,
		}
		data := db.Place{
			ID:       record[0],
			Name:     record[1],
			Address:  record[2],
			Phone:    record[3],
			Location: geo,
		}
		PlacesSlice = append(PlacesSlice, data)
		// id++
	}
	return PlacesSlice, nil
}

func BulkIndexDocuments(es *elasticsearch.Client, PlacesSlice []db.Place) error {
	var countSuccessful uint64
	start := time.Now().UTC()
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:  "places", // The default index name
		Client: es,       // The Elasticsearch client
		// NumWorkers:    numWorkers,       // The number of worker goroutines
		FlushBytes:    5e+6,             // The flush threshold in bytes
		FlushInterval: 30 * time.Second, // The periodic flush interval
	})
	if err != nil {
		log.Fatalf("Error creating the indexer: %s", err)
	}

	for _, a := range PlacesSlice {
		// Prepare the data payload: encode article to JSON
		//
		data, err := json.Marshal(a)
		if err != nil {
			log.Fatalf("Cannot encode place %s: %s", a.ID, err)
		}

		// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
		//
		// Add an item to the BulkIndexer
		//
		err = bi.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				// Action field configures the operation to perform (index, create, delete, update)
				Action: "index",

				// DocumentID is the (optional) document ID
				DocumentID: a.ID,

				// Body is an `io.Reader` with the payload
				Body: bytes.NewReader(data),

				// OnSuccess is called for each successful operation
				OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
					atomic.AddUint64(&countSuccessful, 1)
				},

				// OnFailure is called for each failed operation
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						log.Printf("ERROR: %s", err)
					} else {
						log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
					}
				},
			},
		)
		if err != nil {
			log.Fatalf("Unexpected error: %s", err)
		}
		// <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
	}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	// Close the indexer
	//
	if err := bi.Close(context.Background()); err != nil {
		log.Fatalf("Unexpected error: %s", err)
	}
	// <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	biStats := bi.Stats()

	// Report the results: number of indexed docs, number of errors, duration, indexing rate
	//
	log.Println(strings.Repeat("▔", 65))

	dur := time.Since(start)

	if biStats.NumFailed > 0 {
		log.Fatalf(
			"Indexed [%s] documents with [%s] errors in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			humanize.Comma(int64(biStats.NumFailed)),
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
	} else {
		log.Printf(
			"Sucessfuly indexed [%s] documents in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
	}
	return nil
}

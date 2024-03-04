package main

import (
	dataloader "elasticsearch/internal/dataLoader"
	"log"
)

func main() {
	////////////////////////////////#############Создаем инстанс клиента###################/////////////////
	es, err := dataloader.InitElasticClient()
	if err != nil {
		log.Fatal(err)
	}
	////////////////////////////////#############СОЗДАЕМ INDEX###################/////////////////
	err1 := dataloader.CreateIndex(es)
	if err1 != nil {
		log.Fatal(err1)
	}
	////////////////////////////////#############ДОБАВЛЯЕМ МАППИНГ###################/////////////////
	err2 := dataloader.AddMapping(es)
	if err2 != nil {
		log.Fatal(err2)
	}
	////открываем файлец значит
	//надо будет запихнуть все это в слайс
	PlacesSlice, err := dataloader.ReadCsv()
	if err != nil {
		log.Fatal(err)
	}
	///////////////////////////////////////########BULK INDEXING###################//////////////////////////////////////////////////////////////
	err3 := dataloader.BulkIndexDocuments(es, PlacesSlice)
	if err3 != nil {
		log.Fatal(err3)
	}
}

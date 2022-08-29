package test

import (
	"encoding/csv"
	"log"
	"os"
)

func LoadCSV(filepath string) *csv.Reader {
	csvFile, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(csvFile)
	if _, err := reader.Read(); err != nil {
		log.Fatal(err)
	}

	return reader
}

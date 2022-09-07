package main

import (
	"fmt"
	"log"
	"mssfoobar/ar2-import/ar2-import/kafka"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go"
)

type Config struct {
	minioClient *minio.Client
}

func main() {
	godotenv.Load("../../../.env")

	app := Config{
		minioClient: setMinioClient(),
	}

	go kafka.StartKafka()

	fmt.Println("Kafka has been started...")

	// time.Sleep(10 * time.Minute)

	// app.getObject()

	port := ":" + os.Getenv("APP_PORT")

	err := http.ListenAndServe(port, app.routes())

	if err != nil {
		log.Fatalln(err)
	}

	// var gqlClient test.Activities

	// gqlClient.GraphQlClient = test.TestImportWorkflow()

	// gqlClient.ImportCsvActivity("./lib/data/data.csv")
}

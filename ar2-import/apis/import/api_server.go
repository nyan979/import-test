package main

import (
	"log"
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

	port := ":" + os.Getenv("APP_PORT")

	err := http.ListenAndServe(port, app.routes())

	if err != nil {
		log.Fatalln(err)
	}

	//var gqlClient test.Activities

	//gqlClient.GraphQlClient = test.TestImportWorkflow()

	//gqlClient.ImportCsvActivity("./lib/data/data.csv")
}

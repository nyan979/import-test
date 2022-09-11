package main

import (
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"net/http"
	"os"

	"github.com/hasura/go-graphql-client"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go"
)

type Application struct {
	minioClient   *minio.Client
	graphqlClient *graphql.Client
}

func main() {
	godotenv.Load("../../../.env")

	app := Application{
		minioClient:   utils.SetMinioClient(),
		graphqlClient: utils.SetGraphqlClient(),
	}

	// go kafka.StartKafka()

	// fmt.Println("Kafka has been started...")

	// time.Sleep(10 * time.Minute)

	// app.getObject()

	port := ":" + os.Getenv("APP_PORT")

	err := http.ListenAndServe(port, app.routes())
	if err != nil {
		log.Fatalln(err)
	}

	// var activities test.Activities

	// activities.GraphQlClient = app.graphqlClient

	// config, err := activities.ReadConfigTable("serviceX")
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// fmt.Println(config[0].FileKey)
	// gqlClient.ImportCsvActivity("./lib/data/data.csv")
}

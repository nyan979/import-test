package main

import (
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"

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

	// go app.StartKafka()

	// fmt.Println("Kafka has been started...")

	// time.Sleep(10 * time.Minute)

	// port := ":" + os.Getenv("APP_PORT")

	// err := http.ListenAndServe(port, app.routes())
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	var activities workflow.Activities

	activities.GraphQlClient = app.graphqlClient
	activities.MinioClient = app.minioClient

	activities.GetObject("serviceX-12345")

	// gqlClient.ImportCsvActivity("./lib/data/data.csv")
}

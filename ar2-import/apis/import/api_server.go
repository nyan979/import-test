package main

import (
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"net/http"
	"os"

	"github.com/hasura/go-graphql-client"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go"
	"github.com/segmentio/kafka-go"
)

type Application struct {
	minioClient   *minio.Client
	graphqlClient *graphql.Client
	kafkaReader   *kafka.Reader
	kafkaWriter   *kafka.Writer
}

func main() {
	godotenv.Load("../../../.env")

	app := Application{
		minioClient:   utils.SetMinioClient(),
		graphqlClient: utils.SetGraphqlClient(),
		kafkaReader:   utils.NewKafkaReader(),
		kafkaWriter:   utils.NewKafkaWriter(),
	}

	// var activities workflow.Activities
	// activities.KafkaReader = app.kafkaReader
	// activities.KafkaWriter = app.kafkaWriter

	// ctx := context.Background()
	// message := make(chan kafka.Message, 1000)
	// messageCommitChan := make(chan kafka.Message, 1000)

	// g, ctx := errgroup.WithContext(ctx)

	// g.Go(func() error {
	// 	return activities.ReadMessage(ctx, message)
	// })

	// g.Go(func() error {
	// 	return activities.WriteMessages(ctx, message, messageCommitChan)
	// })

	// g.Go(func() error {
	// 	return activities.CommitMessages(ctx, messageCommitChan)
	// })

	// err := g.Wait()
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// g.Go(func() error {
	// 	return app.StartKafkaConsumer(ctx, message)
	// })

	// go app.StartKafkaProducer()

	// fmt.Println("Kafka has been started...")

	// time.Sleep(10 * time.Minute)

	// var activities workflow.Activities

	// activities.GraphQlClient = app.graphqlClient
	// activities.MinioClient = app.minioClient

	// activities.ParseLine("serviceX-12345")

	// gqlClient.ImportCsvActivity("./lib/data/data.csv")

	port := ":" + os.Getenv("APP_PORT")

	err := http.ListenAndServe(port, app.routes())
	if err != nil {
		log.Fatalln(err)
	}
}

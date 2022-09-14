package main

import (
	"context"
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"
	"net/http"
	"os"

	"github.com/hasura/go-graphql-client"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go"
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"
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

	var activities workflow.Activities
	activities.MinioClient = app.minioClient
	activities.GraphQlClient = app.graphqlClient
	activities.KafkaReader = app.kafkaReader
	activities.KafkaWriter = app.kafkaWriter

	ctx := context.Background()
	message := make(chan kafka.Message, 1000)
	messageCommit := make(chan kafka.Message, 1000)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return activities.FetchMessage(ctx, message)
	})

	g.Go(func() error {
		return activities.WriteMessages(ctx, message, messageCommit)
	})

	g.Go(func() error {
		return activities.CommitMessages(ctx, messageCommit)
	})

	port := ":" + os.Getenv("APP_PORT")

	err := http.ListenAndServe(port, app.routes())
	if err != nil {
		log.Fatalln(err)
	}

	conErr := g.Wait()
	if conErr != nil {
		log.Fatalln(conErr)
	}
}

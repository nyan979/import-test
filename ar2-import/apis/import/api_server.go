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
}

var RequestId chan string

func main() {
	godotenv.Load("../../../.env")

	app := Application{
		minioClient:   utils.SetMinioClient(),
		graphqlClient: utils.SetGraphqlClient(),
		kafkaReader:   utils.NewKafkaReader(),
	}

	activities := workflow.Activities{
		MinioClient:   app.minioClient,
		GraphQlClient: app.graphqlClient,
		KafkaReader:   app.kafkaReader,
	}

	ctx := context.Background()
	RequestId = make(chan string, 1000)
	message := make(chan kafka.Message, 1000)
	messageCommit := make(chan kafka.Message, 1000)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return activities.FetchMessage(ctx, message)
	})

	g.Go(func() error {
		return activities.WriteMessages(ctx, message, messageCommit, RequestId)
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

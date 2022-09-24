package main

import (
	"context"
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	"go.temporal.io/sdk/client"
	"golang.org/x/sync/errgroup"
	zapadapter "logur.dev/adapter/zap"
	"logur.dev/logur"
)

type Application struct {
	temporalClient client.Client
	KafkaReader    *kafka.Reader
	KafkaWriter    *kafka.Writer
	activities     workflow.Activities
}

func main() {
	godotenv.Load("../../../.env")

	logger := logur.LoggerToKV(zapadapter.New(utils.InitZapLogger()))

	client, err := utils.InitTemporalConnection(logger)
	if err != nil {
		logger.Error(err.Error())
	}

	defer client.Close()

	// TODO: implement temporal workflow
	app := Application{
		temporalClient: client,
		KafkaReader:    utils.NewKafkaReader(),
		KafkaWriter:    utils.NewKafkaWriter(),
		activities: workflow.Activities{
			MinioClient:   utils.SetMinioClient(),
			GraphqlClient: utils.SetGraphqlClient(),
			// RequestId:     "",
		},
	}

	ctx := context.Background()
	// RequestId = make(chan string, 1000)
	message := make(chan kafka.Message, 1000)
	messageCommit := make(chan kafka.Message, 1000)

	g, ctx := errgroup.WithContext(ctx)

	// fetch minio notification message go routine
	g.Go(func() error {
		return app.activities.FetchMessage(ctx, message)
	})

	// write csv content to kafka topic go routine
	g.Go(func() error {
		return app.activities.WriteMessages(ctx, message, messageCommit /*, RequestId*/)
	})

	// commit to offset minio notification messages go routine
	g.Go(func() error {
		return app.activities.CommitMessages(ctx, messageCommit)
	})

	// set and serve on port
	port := ":" + os.Getenv("APP_PORT")

	err = http.ListenAndServe(port, app.routes())
	if err != nil {
		log.Fatalln(err)
	}

	conErr := g.Wait()
	if conErr != nil {
		log.Fatalln(conErr)
	}
}

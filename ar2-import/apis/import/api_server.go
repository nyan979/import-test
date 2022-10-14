package main

import (
	"context"
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
	temporalClient    client.Client
	KafkaMinioReader  *kafka.Reader
	KafkaImportReader *kafka.Reader
	KafkaWriter       *kafka.Writer
	activities        workflow.Activities
}

func main() {
	godotenv.Load("../../../.env")

	logger := logur.LoggerToKV(zapadapter.New(utils.InitZapLogger()))

	client, err := utils.InitTemporalConnection(logger)
	if err != nil {
		logger.Error(err.Error())
	}

	defer client.Close()

	app := Application{
		temporalClient:    client,
		KafkaMinioReader:  utils.InitMinioKafkaReader(logger),
		KafkaImportReader: utils.InitImportKafkaReader(logger),
		KafkaWriter:       utils.InitKafkaWriter(logger),
		activities: workflow.Activities{
			MinioClient:   utils.InitMinioClient(logger),
			GraphqlClient: utils.InitGraphqlClient(logger),
		},
	}

	ctx := context.Background()
	minioMessage := make(chan kafka.Message, 1000)
	minioMessageCommit := make(chan kafka.Message, 1000)
	importMessage := make(chan kafka.Message, 1000)
	importMessageCommit := make(chan kafka.Message, 1000)

	g, ctx := errgroup.WithContext(ctx)

	// fetch minio notification message go routine
	g.Go(func() error {
		return app.FetchMinioMessage(ctx, minioMessage, logger)
	})

	// fetch import notification message go routine
	g.Go(func() error {
		return app.FetchImportMessage(ctx, importMessage, importMessageCommit, logger)
	})

	// write csv content to kafka topic go routine
	g.Go(func() error {
		return app.WriteMessages(ctx, minioMessage, minioMessageCommit, logger)
	})

	// commit to offset minio notification message go routine
	g.Go(func() error {
		return app.CommitMinioMessages(ctx, minioMessageCommit, logger)
	})

	// commit to offset import notification message go routine
	g.Go(func() error {
		return app.CommitImportMessages(ctx, importMessageCommit, logger)
	})

	port := ":" + os.Getenv("APP_PORT")

	err = http.ListenAndServe(port, app.routes(logger))
	if err != nil {
		logger.Error(err.Error())
	}

	goErr := g.Wait()
	if goErr != nil {
		logger.Error(goErr.Error())
	}
}

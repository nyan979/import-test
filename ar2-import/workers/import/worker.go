package main

import (
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/worker"

	zapadapter "logur.dev/adapter/zap"
	"logur.dev/logur"
)

func main() {
	godotenv.Load("../../../.env")

	logger := logur.LoggerToKV(zapadapter.New(utils.InitZapLogger()))
	temporalClient, err := utils.InitTemporalConnection(logger)
	if err != nil {
		logger.Error(err.Error())
	}

	defer temporalClient.Close()

	// workerOptions := worker.Options{
	// 	EnableSessionWorker: true,
	// }

	w := worker.New(temporalClient, "import-service", worker.Options{})

	w.RegisterWorkflow(workflow.ImportServiceWorkflow)
	w.RegisterWorkflow(workflow.SignalImportServiceWorkflow)
	w.RegisterActivity(&workflow.Activities{
		MinioClient:   utils.SetMinioClient(),
		GraphqlClient: utils.SetGraphqlClient(),
		KafkaReader:   utils.NewKafkaReader(),
		KafkaWriter:   utils.NewKafkaWriter(),
	})

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Error("Unable to start workflow: " + err.Error())
	}
}

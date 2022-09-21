package main

import (
	"mssfoobar/ar2-import/ar2-import/lib/utils"

	"go.temporal.io/sdk/worker"

	zapadapter "logur.dev/adapter/zap"
	"logur.dev/logur"
)

func main() {
	logger := logur.LoggerToKV(zapadapter.New(utils.InitZapLogger()))
	temporalClient, err := utils.InitTemporalConnection(logger)
	if err != nil {
		logger.Error(err.Error())
	}

	defer temporalClient.Close()

	workerOptions := worker.Options{
		EnableSessionWorker: true,
	}

	w := worker.New(temporalClient, "import-service", workerOptions)

	// w.RegisterWorkflow()

	//w := worker.New(temporalClient, "Get-PresignedUrl", worker.Options{})

	// TO CONTINUE
}

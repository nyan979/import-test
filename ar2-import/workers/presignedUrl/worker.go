package presignedUrl

import (
	"mssfoobar/ar2-import/ar2-import/lib/utils"

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

	//w := worker.New(temporalClient, "Get-PresignedUrl", worker.Options{})

	// TO CONTINUE
}

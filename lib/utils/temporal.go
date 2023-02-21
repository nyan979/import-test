package utils

import (
	"os"

	"go.temporal.io/sdk/client"
	"logur.dev/logur"
)

type TemporalConf struct {
	Endpoint  string
	Namespace string
}

func GetTemporalClient(conf TemporalConf, logger logur.KVLoggerFacade) *client.Client {
	logger.Info("--- Connecting to Temporal ---")
	client, err := client.Dial(client.Options{
		HostPort:  conf.Endpoint,
		Namespace: conf.Namespace,
		Logger:    logger,
	})
	if err != nil {
		logger.Error("Unable to connect to temporal: %s\n", err)
		os.Exit(1)
	}
	logger.Info("Initialized Temporal Client:", "temporalEndpoint",
		conf.Endpoint, "temporalNamespace", conf.Namespace)
	return &client
}

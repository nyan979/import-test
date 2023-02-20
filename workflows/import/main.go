package main

import (
	"log"
	"mssfoobar/ar2-import/lib/utils"
	"mssfoobar/ar2-import/workflows/import/workflow"
	"os"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/worker"

	zapadapter "logur.dev/adapter/zap"
	"logur.dev/logur"

	graphqlService "mssfoobar/ar2-import/apps/graphql"
	minioService "mssfoobar/ar2-import/apps/minio"
	temporalService "mssfoobar/ar2-import/apps/temporal"
)

type Config struct {
	logLevel string
	minio    minioService.MinioConf
	graphql  graphqlService.GraphqlConf
	temporal temporalService.TemporalConf
}

func (c *Config) load() error {
	err := godotenv.Load("../../.env")
	if err != nil {
		return err
	}
	c.logLevel = os.Getenv("LOG_LEVEL")
	c.minio = minioService.MinioConf{
		Endpoint:        os.Getenv("MINIO_ENDPOINT"),
		UseSSL:          false,
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
		BucketName:      os.Getenv("MINIO_BUCKET_NAME"),
		HostName:        os.Getenv("MINIO_HOST_NAME"),
	}
	c.graphql = graphqlService.GraphqlConf{
		HasuraAddress:  os.Getenv("HASURA_ADDRESS"),
		GraphqlEndpint: os.Getenv("GQL_ENDPOINT"),
		AdminKey:       os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET"),
	}
	c.temporal = temporalService.TemporalConf{
		Endpoint:  os.Getenv("TEMPORAL_ENDPOINT"),
		Namespace: os.Getenv("TEMPORAL_NAMESPACE"),
	}
	return nil
}

func main() {
	var conf Config
	err := conf.load()
	if err != nil {
		log.Fatal("Environment file loading failed.", err)
	}
	logger := logur.LoggerToKV(zapadapter.New(utils.InitZapLogger(conf.logLevel)))
	temporalClient := *temporalService.GetTemporalClient(conf.temporal, logger)
	if err != nil {
		logger.Error(err.Error())
	}
	defer temporalClient.Close()
	w := worker.New(temporalClient, "import-service", worker.Options{})
	w.RegisterWorkflow(workflow.ImportServiceWorkflow)
	w.RegisterWorkflow(workflow.SignalImportServiceWorkflow)
	w.RegisterActivity(&workflow.Activities{
		MinioClient:   minioService.GetMinioClient(conf.minio, logger),
		GraphqlClient: graphqlService.GetGraphqlClient(conf.graphql, logger),
	})

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Error("Unable to start workflow: " + err.Error())
	}
}

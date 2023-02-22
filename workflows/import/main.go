package main

import (
	"flag"
	"mssfoobar/ar2-import/lib/utils"
	"mssfoobar/ar2-import/workflows/import/workflow"
	"os"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/worker"

	zapadapter "logur.dev/adapter/zap"
	"logur.dev/logur"
)

type Config struct {
	logLevel string
	minio    utils.MinioConf
	graphql  utils.GraphqlConf
	temporal utils.TemporalConf
}

func (c *Config) load(confFile string) {
	godotenv.Load(confFile)
	c.logLevel = os.Getenv("LOG_LEVEL")
	c.minio = utils.MinioConf{
		Endpoint:        os.Getenv("MINIO_HOST") + ":" + os.Getenv("MINIO_PORT"),
		UseSSL:          false,
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
		BucketName:      os.Getenv("MINIO_BUCKET_NAME"),
		PublicHostName:  os.Getenv("MINIO_HOST_PUBLIC"),
	}
	c.graphql = utils.GraphqlConf{
		HasuraAddress:  os.Getenv("HASURA_HOST") + ":" + os.Getenv("HASURA_PORT"),
		GraphqlEndpint: os.Getenv("GQL_ENDPOINT"),
		AdminKey:       os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET"),
	}
	c.temporal = utils.TemporalConf{
		Endpoint:  os.Getenv("TEMPORAL_HOST") + ":" + os.Getenv("TEMPORAL_PORT"),
		Namespace: os.Getenv("TEMPORAL_NAMESPACE"),
	}
}

func main() {
	var confFile string
	flag.StringVar(&confFile, "c", "../../.env", "environment file")
	flag.Parse()

	if confFile == "" {
		flag.PrintDefaults()
		return
	}
	var conf Config
	conf.load(confFile)

	logger := logur.LoggerToKV(zapadapter.New(utils.InitZapLogger(conf.logLevel)))
	temporalClient := *utils.GetTemporalClient(conf.temporal, logger)
	defer temporalClient.Close()

	w := worker.New(temporalClient, "import-service", worker.Options{})
	w.RegisterWorkflow(workflow.ImportServiceWorkflow)
	w.RegisterWorkflow(workflow.SignalImportServiceWorkflow)
	w.RegisterActivity(&workflow.Activities{
		MinioClient:   utils.GetMinioClient(conf.minio, logger),
		GraphqlClient: utils.GetGraphqlClient(conf.graphql, logger),
	})

	err := w.Run(worker.InterruptCh())
	if err != nil {
		logger.Error("Unable to start workflow: " + err.Error())
	}
}

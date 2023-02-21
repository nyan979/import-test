package server

import (
	"log"
	"mssfoobar/ar2-import/lib/utils"
	"mssfoobar/ar2-import/workflows/import/workflow"
	"net/http"
	"os"
	"time"

	graphqlService "mssfoobar/ar2-import/apps/graphql"
	minioService "mssfoobar/ar2-import/apps/minio"
	temporalService "mssfoobar/ar2-import/apps/temporal"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	zapadapter "logur.dev/adapter/zap"
	"logur.dev/logur"
)

type ImportService struct {
	timeLive       string
	timeReady      string
	temporalClient *client.Client
	activities     workflow.Activities
	logger         *logur.KVLoggerFacade
	conf           Config
}

type Config struct {
	port     string
	logLevel string
	minio    minioService.MinioConf
	graphql  graphqlService.GraphqlConf
	temporal temporalService.TemporalConf
}

func New() *ImportService {
	return &ImportService{}
}

func (c *Config) Load() error {
	err := godotenv.Load("../../.env")
	if err != nil {
		return err
	}
	c.port = os.Getenv("APP_PORT")
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

func (srv ImportService) Start(conf Config) {
	timeLive := time.Now().Format(time.RFC3339)
	logger := logur.LoggerToKV(zapadapter.New(utils.InitZapLogger(conf.logLevel)))
	client := *temporalService.GetTemporalClient(conf.temporal, logger)
	defer client.Close()
	srv = ImportService{
		temporalClient: &client,
		timeLive:       timeLive,
		timeReady:      time.Now().Format(time.RFC3339),
		activities: workflow.Activities{
			MinioClient:   minioService.GetMinioClient(conf.minio, logger),
			GraphqlClient: graphqlService.GetGraphqlClient(conf.graphql, logger),
		},
		logger: &logger,
	}
	logger.Info("HTTP Server Starting On Port:", "Port", conf.port)
	port := ":" + conf.port
	err := http.ListenAndServe(port, srv.routes(logger))
	if err != nil {
		log.Fatal("Failed to start server.", err)
	}
}

// Deprecated
/*
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
goErr := g.Wait()
if goErr != nil {
	logger.Error(goErr.Error())
}
*/

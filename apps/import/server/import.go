package server

import (
	"log"
	"mssfoobar/ar2-import/lib/utils"
	"mssfoobar/ar2-import/workflows/import/workflow"
	"net/http"
	"os"
	"time"

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
	minio    utils.MinioConf
	graphql  utils.GraphqlConf
	temporal utils.TemporalConf
	nats     utils.NatsConf
}

func New() *ImportService {
	return &ImportService{}
}

func (c *Config) Load(confFile string) {
	godotenv.Load(confFile)
	c.port = os.Getenv("APP_PORT")
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
	c.nats = utils.NatsConf{
		URL:     os.Getenv("NATS_URL"),
		Subject: os.Getenv("NATS_SUBJECT"),
	}
	log.Println(c)
}

func (srv ImportService) Start(conf Config) {
	timeLive := time.Now().Format(time.RFC3339)
	logger := logur.LoggerToKV(zapadapter.New(utils.InitZapLogger(conf.logLevel)))

	natsConn, err := utils.NewNatsConn(conf.nats, logger)
	if err != nil {
		logger.Error("Nats connection error", "Error", err)
		os.Exit(1)
	}

	updateFileStatus, err := natsConn.Subscribe(conf.nats.Subject, srv.updateFileStatus)
	if err != nil {
		logger.Error("Nats subscription error", "Error", err)
		os.Exit(1)
	}
	defer updateFileStatus.Unsubscribe()

	temporalClient := *utils.GetTemporalClient(conf.temporal, logger)
	defer temporalClient.Close()

	srv = ImportService{
		temporalClient: &temporalClient,
		timeLive:       timeLive,
		timeReady:      time.Now().Format(time.RFC3339),
		activities: workflow.Activities{
			MinioClient:   utils.GetMinioClient(conf.minio, logger),
			GraphqlClient: utils.GetGraphqlClient(conf.graphql, logger),
		},
		logger: &logger,
	}
	logger.Info("HTTP Server Starting On Port:", "Port", conf.port)
	port := ":" + conf.port
	err = http.ListenAndServe(port, srv.routes(logger))
	if err != nil {
		logger.Error("Failed to start server.", "Error", err)
		os.Exit(1)
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

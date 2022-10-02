package utils

import (
	"context"
	"fmt"
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"
	"net/http"
	"os"

	"github.com/hasura/go-graphql-client"
	"github.com/minio/minio-go"
	"github.com/segmentio/kafka-go"
	"go.temporal.io/sdk/client"
	temporalLog "go.temporal.io/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func SetMinioClient() *minio.Client {
	host := os.Getenv("MINIO_HOST")
	port := os.Getenv("MINIO_PORT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := false

	endpoint := host + ":" + port

	client, err := minio.New(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		log.Fatalln(err)
	}

	return client
}

func SetGraphqlClient() *graphql.Client {
	dbHost := os.Getenv("HASURA_HOST")
	dbPort := os.Getenv("HASURA_PORT")
	gqlEndpoint := os.Getenv("GQL_ENDPOINT")
	adminkey := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")

	graphqlURL := "http://" + dbHost + ":" + dbPort + "/" + gqlEndpoint

	client := graphql.NewClient(graphqlURL, nil)
	client = client.WithRequestModifier(func(req *http.Request) {
		req.Header.Set("x-hasura-admin-secret", adminkey)
	})

	return client
}

func NewMinioKafkaReader() *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{os.Getenv("KAFKA_HOST") + ":" + os.Getenv("KAFKA_PORT")},
		Topic:   os.Getenv("KAFKA_IMPORT_TOPIC"),
		GroupID: "testing",
	})

	return reader
}

func NewImportKafkaReader() *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{os.Getenv("KAFKA_HOST") + ":" + os.Getenv("KAFKA_PORT")},
		Topic:   os.Getenv("KAFKA_COMPLETE_TOPIC"),
		GroupID: "testing",
	})

	return reader
}

func NewKafkaWriter() *kafka.Writer {

	writer := &kafka.Writer{
		Addr:  kafka.TCP(os.Getenv("KAFKA_HOST") + ":" + os.Getenv("KAFKA_PORT")),
		Topic: os.Getenv("KAFKA_SERVICE_TOPIC"),
	}

	return writer
}

func InitZapLogger() *zap.Logger {

	isDevelopment := os.Getenv("DEVELOPMENT")

	// Default to production level
	logLevel := zap.InfoLevel
	isDev := false

	// Set development to TRUE if DEVELOPMENT is set to true, otherwise default to false
	if len(isDevelopment) != 0 {
		if isDevelopment == "true" {
			isDev = true
			logLevel = zap.DebugLevel
		}
	}

	encodeConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(logLevel),
		Development:      isDev,
		Sampling:         nil, // consider exposing this to config
		Encoding:         "console",
		EncoderConfig:    encodeConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()

	if err != nil {
		log.Fatal("Unable to create zap logger")
	}

	return logger
}

func InitTemporalConnection(logger temporalLog.Logger) (client.Client, error) {
	var TemporalHost = "localhost"
	var TemporalPort = "7233"
	var TemporalNamespace = "default"

	logger.Info("Temporal Connection Details:", "temporalHost", TemporalHost, "temporalPort", TemporalPort, "temporalNamespace", TemporalNamespace)

	return client.Dial(client.Options{
		HostPort:  fmt.Sprintf("%v:%v", TemporalHost, TemporalPort),
		Namespace: TemporalNamespace,
		Logger:    logger,
	})
}

func CreateImportWorkflow(c client.Client, status *workflow.ImportStatus) error {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "import-service",
		ID:        status.Message.RequestID,
	}
	ctx := context.Background()

	_, err := c.ExecuteWorkflow(ctx, workflowOptions, workflow.ImportServiceWorkflow)
	if err != nil {
		return fmt.Errorf("unable to execute order workflow: %w", err)
	}

	return nil
}

func ExecuteImportWorkflow(c client.Client, requestId string, status *workflow.ImportStatus) (*workflow.ImportStatus, error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "import-service",
	}
	ctx := context.Background()

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workflow.SignalImportServiceWorkflow, requestId, status)
	if err != nil {
		return status, fmt.Errorf("unable to execute workflow: %w", err)
	}
	err = we.Get(ctx, &status)
	if err != nil {
		return status, err
	}

	return status, nil
}

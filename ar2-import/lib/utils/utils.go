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

func InitMinioClient(logger temporalLog.Logger) *minio.Client {
	host := os.Getenv("MINIO_HOST")
	port := os.Getenv("MINIO_PORT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := false

	endpoint := host + ":" + port

	client, err := minio.New(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		logger.Error(err.Error())
	}

	logger.Info("Initialized Minio Client:", "minioHost", host, "minioPort", port)

	return client
}

func InitGraphqlClient(logger temporalLog.Logger) *graphql.Client {
	host := os.Getenv("HASURA_HOST")
	port := os.Getenv("HASURA_PORT")
	gqlEndpoint := os.Getenv("GQL_ENDPOINT")
	adminkey := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")

	graphqlURL := "http://" + host + ":" + port + "/" + gqlEndpoint

	client := graphql.NewClient(graphqlURL, nil)
	client = client.WithRequestModifier(func(req *http.Request) {
		req.Header.Set("x-hasura-admin-secret", adminkey)
	})

	logger.Info("Initialized Hasura GraphQL Client:", "hasuraHost", host, "hasuraPort", port, "gqlEndpoint", gqlEndpoint)

	return client
}

func InitMinioKafkaReader(logger temporalLog.Logger) *kafka.Reader {
	host := os.Getenv("KAFKA_HOST")
	port := os.Getenv("KAFKA_PORT")
	topic := os.Getenv("KAFKA_MINIO_TOPIC")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{host + ":" + port},
		Topic:   topic,
		GroupID: "minio-consumer-group-1",
	})

	logger.Info("Initialized Minio Kafka Reader:", "brokerHost", host, "brokerPort", port, "topic", topic)

	return reader
}

func InitImportKafkaReader(logger temporalLog.Logger) *kafka.Reader {
	host := os.Getenv("KAFKA_HOST")
	port := os.Getenv("KAFKA_PORT")
	topic := os.Getenv("KAFKA_SERVICE_OUT_TOPIC")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{host + ":" + port},
		Topic:   topic,
		GroupID: "import-consumer-group-1",
	})

	logger.Info("Initialized Service_Out Kafka Reader:", "brokerHost", host, "brokerPort", port, "topic", topic)

	return reader
}

func InitKafkaWriter(logger temporalLog.Logger) *kafka.Writer {
	host := os.Getenv("KAFKA_HOST")
	port := os.Getenv("KAFKA_PORT")
	topic := os.Getenv("KAFKA_SERVICE_IN_TOPIC")

	writer := &kafka.Writer{
		Addr:  kafka.TCP(host + ":" + port),
		Topic: topic,
	}

	logger.Info("Initialized Service_In Kafka Writer:", "brokerHost", host, "brokerPort", port, "topic", topic)

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
	var TemporalHost = os.Getenv("TEMPORAL_HOST")
	var TemporalPort = os.Getenv("TEMPORAL_PORT")
	var TemporalNamespace = os.Getenv("TEMPORAL_NAMESPACE")

	logger.Info("Temporal Connection Details:", "temporalHost", TemporalHost, "temporalPort", TemporalPort, "temporalNamespace", TemporalNamespace)

	return client.Dial(client.Options{
		HostPort:  fmt.Sprintf("%v:%v", TemporalHost, TemporalPort),
		Namespace: TemporalNamespace,
		Logger:    logger,
	})
}

func CreateImportWorkflow(c client.Client, status *workflow.ImportSignal) error {
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

func ExecuteImportWorkflow(c client.Client, requestId string, status *workflow.ImportSignal) (*workflow.ImportSignal, error) {
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

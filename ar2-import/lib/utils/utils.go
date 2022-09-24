package utils

import (
	"fmt"
	"log"
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

func NewKafkaReader() *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{os.Getenv("KAFKA_HOST") + ":" + os.Getenv("KAFKA_PORT")},
		Topic:   os.Getenv("KAFKA_IMPORT_TOPIC"),
		GroupID: "minio-consumer-group-1",
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

	// TemporalHost := os.Getenv("TEMPORAL_HOST")

	// if len(TemporalHost) == 0 {
	// 	TemporalHost = TemporalHostDefault
	// 	logger.Warn("TEMPORAL_HOST not set, using defaults", "TemporalHost", TemporalHost)
	// }

	// TemporalPort := os.Getenv("TEMPORAL_GRPC_PORT")

	// if len(TemporalPort) == 0 {
	// 	TemporalPort = TemporalPortDefault
	// 	logger.Warn("TEMPORAL_PORT not set, using defaults", "TemporalPort", TemporalPort)
	// }

	// TemporalNamespace := os.Getenv("TEMPORAL_NAMESPACE")

	// if len(TemporalNamespace) == 0 {
	// 	TemporalNamespace = TemporalNamespaceDefault
	// 	logger.Warn("TEMPORAL_NAMESPACE not set, using defaults", "TEMPORAL_NAMESPACE", TemporalNamespace)
	// }

	logger.Info("Temporal Connection Details:", "temporalHost", TemporalHost, "temporalPort", TemporalPort, "temporalNamespace", TemporalNamespace)

	return client.Dial(client.Options{
		HostPort:  fmt.Sprintf("%v:%v", TemporalHost, TemporalPort),
		Namespace: TemporalNamespace,
		Logger:    logger,
	})
}

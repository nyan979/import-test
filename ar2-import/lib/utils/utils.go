package utils

import (
	"log"
	"net/http"
	"os"

	"github.com/hasura/go-graphql-client"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func SetMinioClient() *minio.Client {
	host := os.Getenv("MINIO_HOST")
	port := os.Getenv("MINIO_PORT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := true

	endpoint := host + ":" + port

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	return minioClient
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

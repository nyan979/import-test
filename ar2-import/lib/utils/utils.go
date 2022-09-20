package utils

import (
	"log"
	"net/http"
	"os"

	"github.com/hasura/go-graphql-client"
	"github.com/minio/minio-go"
	"github.com/segmentio/kafka-go"
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

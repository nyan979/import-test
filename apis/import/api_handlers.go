package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/minio/minio-go"
)

type MinioClient struct {
	client *minio.Client
}

func setMinioClient() *minio.Client {
	host := os.Getenv("MINIO_HOST")
	port := os.Getenv("MINIO_PORT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := false

	endpoint := host + ":" + port

	// Initialize minio client object.
	client, err := minio.New(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		log.Fatalln(err)
	}

	return client
}

func (m *MinioClient) getPresignedUrl(w http.ResponseWriter, r *http.Request) {
	presignedURL, err := m.client.PresignedPutObject("ar2-import-bucket", "sample", time.Duration(1000)*time.Second)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(presignedURL)
}

func (m *MinioClient) SetupRoutes() {
	port := ":" + os.Getenv("APP_PORT")
	http.HandleFunc("/upload", m.getPresignedUrl)
	http.ListenAndServe(port, nil)
}

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/minio/minio-go"
)

type MinioClient struct {
	client *minio.Client
}

func setMinioClient() *minio.Client {
	endpoint := "localhost:9000"
	accessKeyID := "minioadmin"
	secretAccessKey := "minioadmin"
	useSSL := false

	// Initialize minio client object.
	client, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
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

// func (m *MinioClient) listenBucketNotification() {
// 	m.client.ListenBucketNotification("ar2-import-bucket", "csv-files/", ".csv", []string {

// 	})
// }

func (m *MinioClient) SetupRoutes() {
	http.HandleFunc("/upload", m.getPresignedUrl)
	http.ListenAndServe(":5000", nil)
}

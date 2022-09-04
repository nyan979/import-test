package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/minio/minio-go"
)

// type RequestPayload struct {
// 	RequestId  string `json:"requestId"`
// 	UploadType string `json:"uploadType`
// }

type jsonResponse struct {
	Url string `json:"url"`
}

func setMinioClient() *minio.Client {
	host := os.Getenv("MINIO_HOST")
	port := os.Getenv("MINIO_PORT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := false

	// host := "localhost"
	// port := "9000"
	// accessKey := "minioadmin"
	// secretKey := "minioadmin"
	// useSSL := false

	endpoint := host + ":" + port

	// Initialize minio client object.
	client, err := minio.New(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		log.Fatalln(err)
	}

	return client
}

func (app *Config) getPresignedUrl(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	presignedURL, err := app.minioClient.PresignedPutObject("ar2-import-bucket", "sample", time.Duration(1000)*time.Second)
	if err != nil {
		log.Fatalln(err)
	}

	payload := jsonResponse{
		Url: presignedURL.String(),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Fatalln(err)
	}

	// log.Println(presignedURL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(jsonPayload)
}

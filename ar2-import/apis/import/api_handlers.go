package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/minio/minio-go"
)

type jsonResponse struct {
	PresignedUrl string `json:"presignedUrl"`
	RequestId    string `json:"requestId"`
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

func (app *Config) getVersionInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	buildVersion := os.Getenv("SERVICE_VERSION")
	w.Header().Set("Content-Type", "application/json")
	if len(buildVersion) == 0 {
		buildVersion = "unknown"
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte(buildVersion))
}

func (app *Config) getHealthInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}

func (app *Config) getPresignedUrl(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	presignedURL, err := app.minioClient.PresignedPutObject("ar2-import", "sample", time.Duration(300)*time.Second)
	if err != nil {
		log.Fatalln(err)
		return
	}

	uploadType := ps.ByName("uploadType")
	requestId := ps.ByName("requestId")
	if len(requestId) == 0 || len(uploadType) == 0 {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payload := jsonResponse{
		PresignedUrl: presignedURL.String(),
		RequestId:    requestId,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Fatalln(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonPayload)
}

func (app *Config) getObject() {
	object, err := app.minioClient.GetObject("ar2-import", "data.csv", minio.GetObjectOptions{})
	if err != nil {
		log.Fatalln(err)
		return
	}

	reader := csv.NewReader(object)

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		log.Println(line)
	}
}

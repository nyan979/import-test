package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	"github.com/minio/minio-go"
)

type jsonResponse struct {
	PresignedUrl string `json:"presignedUrl"`
	RequestId    string `json:"requestId"`
}

func (app *Application) getVersionInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

func (app *Application) getHealthInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}

func (app *Application) getPresignedUrl(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	uploadType := ps.ByName("uploadType")
	requestId := ps.ByName("requestId")

	if len(requestId) == 0 || len(uploadType) == 0 {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	var activities workflow.Activities
	activities.GraphQlClient = app.graphqlClient
	activities.MinioClient = app.minioClient

	config, err := activities.ReadConfigTable("serviceX")
	if err != nil {
		log.Fatalln(err)
		return
	}

	url, err := activities.GetPresignedUrl(config)
	if err != nil {
		log.Fatalln(err)
		return
	}

	payload := jsonResponse{
		PresignedUrl: url.String(),
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

func (app *Application) getObject() {
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

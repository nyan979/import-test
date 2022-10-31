package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
)

// json payload for presigned url response
type jsonResponse struct {
	PresignedUrl string `json:"presignedUrl"`
	RequestId    string `json:"requestId"`
}

// service version end point
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

	// log.Println("Inside Get Version Info")
	// log.Println(r)
}

// service health end point
func (app *Application) getHealthInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))

	// log.Println("Inside Get Health Info")
	// log.Println(r)
}

// to recieve requestId and response with presigned Url
func (app *Application) getPresignedUrl(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	uploadType := ps.ByName("uploadType")
	requestId := ps.ByName("requestId")

	// log.Println("Inside PresignedURL")
	// log.Println(r)

	// invalid request on empty http parameter
	if requestId == ":requestId" || uploadType == ":uploadType" {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	// minio client call to get presigned url and response back to browser

	presignedURL, err := app.activities.MinioClient.PresignedPutObject(context.Background(),
		os.Getenv("MINIO_BUCKET_NAME"), "testUpload.csv", time.Duration(300)*time.Second)
	if err != nil {
		return
	}

	payload := jsonResponse{
		PresignedUrl: presignedURL.String(),
		RequestId:    requestId,
	}

	jsonPayload, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonPayload)
}

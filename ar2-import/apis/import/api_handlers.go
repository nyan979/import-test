package main

import (
	"encoding/json"
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"
	"net/http"
	"os"

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

	// invalid request on empty http parameter
	if requestId == ":requestId" || uploadType == ":uploadType" {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	status := &workflow.ImportStatus{
		Message: workflow.ImportMessage{
			RequestID: requestId, UploadType: uploadType,
		},
	}

	status, err := utils.CreateImportWorkflow(app.temporalClient, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err = utils.UpdateWorkflow(app.temporalClient, requestId, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if status.Stage == "Service not available" {
		log.Println("Inside another upload running")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		payload := jsonResponse{
			RequestId: status.Message.RequestID,
		}
		jsonPayload, _ := json.Marshal(payload)
		w.Write(jsonPayload)
		return
	}

	status, err = utils.UpdateWorkflow(app.temporalClient, requestId, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if status.Stage == "Upload type config not found" {
		http.Error(w, "No such upload type configuration", http.StatusBadRequest)
		return
	}

	status, err = utils.UpdateWorkflow(app.temporalClient, requestId, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var url string

	if status.Stage == "Presigned Url" {
		url = status.Message.URL
	}

	payload := jsonResponse{
		PresignedUrl: url,
		RequestId:    requestId,
	}

	jsonPayload, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonPayload)
}

package main

import (
	"encoding/json"
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
}

// service health end point
func (app *Application) getHealthInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
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

	signal := &workflow.ImportSignal{
		Message: workflow.ImportMessage{
			RequestID: requestId, UploadType: uploadType,
		},
	}

	err := utils.CreateImportWorkflow(app.temporalClient, signal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	signal, err = utils.ExecuteImportWorkflow(app.temporalClient, requestId, signal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var payload any

	switch signal.Stage {
	case workflow.UploadTypeStage:
		http.Error(w, "No such upload type configuration", http.StatusBadRequest)
		return
	case workflow.ServiceBusyStage:
		payload = jsonResponse{
			RequestId: signal.Message.RequestID,
		}
	case workflow.PresignedUrlStage:
		payload = jsonResponse{
			PresignedUrl: signal.Message.URL,
			RequestId:    requestId,
		}
	}

	jsonPayload, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonPayload)
}

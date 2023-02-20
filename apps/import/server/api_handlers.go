package server

import (
	"encoding/json"
	"mssfoobar/ar2-import/lib/utils"
	"mssfoobar/ar2-import/workflows/import/workflow"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// json payload for presigned url response
type jsonResponse struct {
	PresignedUrl string `json:"presignedUrl"`
	RequestId    string `json:"requestId,omitempty"`
}

func (app *Application) getLiveness(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Live since " + app.timeLive))
}

func (app *Application) getReadiness(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if app.timeReady == "" {
		http.Error(w, "Server not ready", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready since " + app.timeReady))
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

func (app *Application) upload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	bucket := ps.ByName("bucket")
	objectName := ps.ByName("objectName")
	expire := ps.ByName("expire")
	if bucket == "" || objectName == "" || expire == "" {
		http.Error(w, "invalid parameter", http.StatusBadRequest)
		return
	}
	duration, err := time.ParseDuration(expire)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := app.activities.PresignedUpload(bucket, duration, objectName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var payload any
	payload = jsonResponse{
		PresignedUrl: url,
	}
	jsonPayload, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonPayload)
}

func (app *Application) download(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	bucket := ps.ByName("bucket")
	objectName := ps.ByName("objectName")
	expire := ps.ByName("expire")
	if bucket == "" || objectName == "" || expire == "" {
		http.Error(w, "invalid parameter", http.StatusBadRequest)
		return
	}
	duration, err := time.ParseDuration(expire)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := app.activities.PresignedDownload(bucket, duration, objectName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var payload any
	payload = jsonResponse{
		PresignedUrl: url,
	}
	jsonPayload, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonPayload)
}

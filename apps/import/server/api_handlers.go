package server

import (
	"encoding/json"
	"log"
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

func (srv *ImportService) getLiveness(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Live since " + srv.timeLive))
}

func (srv *ImportService) getReadiness(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if srv.timeReady == "" {
		http.Error(w, "Server not ready", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready since " + srv.timeReady))
}

// to recieve requestId and response with presigned Url
func (srv *ImportService) getPresignedUrl(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logger := *srv.logger
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
	err := utils.CreateImportWorkflow(*srv.temporalClient, signal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}
	signal, err = utils.ExecuteImportWorkflow(*srv.temporalClient, requestId, signal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var payload any
	switch signal.Stage {
	case workflow.UploadTypeStage:
		logger.Info("Invalid upload type configuration")
		http.Error(w, "Invalid upload type configuration", http.StatusBadRequest)
		return
	case workflow.ServiceBusyStage:
		logger.Info("Service unavailable. Same upload type configuration in progress.")
		payload = jsonResponse{
			RequestId: signal.Message.RequestID,
		}
	case workflow.PresignedUrlStage:
		logger.Info("Presigned url acquired")
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

func (srv *ImportService) upload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logger := *srv.logger
	bucket := ps.ByName("bucket")
	objectName := ps.ByName("objectName")
	expire := ps.ByName("expire")
	if bucket == "" || objectName == "" || expire == "" {
		logger.Info("Invalid path parameter")
		http.Error(w, "invalid parameter", http.StatusBadRequest)
		return
	}
	duration, err := time.ParseDuration(expire)
	if err != nil {
		logger.Info("Invalid expire time. Valid time units are ns, us (or µs), ms, s, m, h.", "Error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := srv.activities.PresignedUpload(bucket, duration, objectName)
	if err != nil {
		logger.Info("Unable to create presigned url", "Error", err)
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

func (srv *ImportService) download(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logger := *srv.logger
	bucket := ps.ByName("bucket")
	objectName := ps.ByName("objectName")
	expire := ps.ByName("expire")
	if bucket == "" || objectName == "" || expire == "" {
		logger.Info("Invalid path parameter")
		http.Error(w, "invalid parameter", http.StatusBadRequest)
		return
	}
	duration, err := time.ParseDuration(expire)
	if err != nil {
		logger.Info("Invalid expire time. Valid time units are ns, us (or µs), ms, s, m, h.", "Error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := srv.activities.PresignedDownload(bucket, duration, objectName)
	if err != nil {
		logger.Info("Unable to create presigned url", "Error", err)
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

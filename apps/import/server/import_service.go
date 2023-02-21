package server

import (
	"context"
	"encoding/json"
	"fmt"
	"mssfoobar/ar2-import/workflows/import/workflow"
	"net/http"
	"strings"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/julienschmidt/httprouter"
	"github.com/nats-io/nats.go"
	"go.temporal.io/sdk/client"
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
	err := createImportWorkflow(*srv.temporalClient, signal)
	if err != nil {
		logger.Error("Cannot create workflow", "Error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	signal, err = executeImportWorkflow(*srv.temporalClient, requestId, signal)
	if err != nil {
		logger.Error("Cannot execute workflow", "Error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var payload any
	switch signal.Stage {
	case workflow.UploadTypeStage:
		logger.Error("Invalid upload type configuration")
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
		logger.Error("Valid time units are ns, us (or µs), ms, s, m, h", "Error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := srv.activities.PresignedUpload(bucket, duration, objectName)
	if err != nil {
		logger.Error("Unable to create presigned url", "Error", err)
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
		logger.Error("Valid time units are ns, us (or µs), ms, s, m, h", "Error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := srv.activities.PresignedDownload(bucket, duration, objectName)
	if err != nil {
		logger.Error("Unable to create presigned url", "Error", err)
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

// initialize temporal import service workflow
func createImportWorkflow(c client.Client, signal *workflow.ImportSignal) error {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "import-service",
		ID:        signal.Message.RequestID,
	}
	ctx := context.Background()
	_, err := c.ExecuteWorkflow(ctx, workflowOptions, workflow.ImportServiceWorkflow)
	if err != nil {
		return fmt.Errorf("Unable to execute order workflow: %w", err)
	}
	return nil
}

// execute external workflow to send/recieve signal channel from/to import service workflow
func executeImportWorkflow(c client.Client, requestId string, signal *workflow.ImportSignal) (*workflow.ImportSignal, error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "import-service",
	}
	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workflow.SignalImportServiceWorkflow, requestId, signal)
	if err != nil {
		return signal, fmt.Errorf("Unable to execute workflow: %w", err)
	}
	err = we.Get(ctx, &signal)
	if err != nil {
		return signal, err
	}
	return signal, nil
}

type jsonMessage struct {
	RequestId string `json:"requestId"`
	Record    string `json:"record"`
}

type MinioMessage struct {
	Key     string `json:"Key"`
	Records []struct {
		S3 struct {
			Object struct {
				Key       string `json:"key"`
				VersionId string `json:"versionId"`
			} `json:"Object"`
		} `json:"s3"`
	} `json:"Records"`
}

func (srv *ImportService) updateFileStatus(msg *nats.Msg) {
	logger := *srv.logger
	var minioMsg MinioMessage
	err := json.Unmarshal(msg.Data, &minioMsg)
	if err != nil {
		return
	}
	if !strings.Contains(minioMsg.Key, "ar2-import/") {
		logger.Info("Minio notification", "Notification", minioMsg.Key)
		return
	}

	signal := &workflow.ImportSignal{}
	signal.Message.FileKey = minioMsg.Records[0].S3.Object.Key
	signal.Message.FileVersionID = minioMsg.Records[0].S3.Object.VersionId

	signal.Message.RequestID, err = srv.getRequestIdByFileKey(signal.Message.FileKey)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	signal, err = executeImportWorkflow(*srv.temporalClient, signal.Message.RequestID, signal)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	var jsonMessages []jsonMessage
	for _, line := range signal.Message.Record {
		jsonMessages = append(jsonMessages, jsonMessage{
			RequestId: string(signal.Message.RequestID),
			Record:    line,
		})
	}
	logger.Info("Parsed CSV", "Records", jsonMessages)
}

// query to get requestId by file name
func (srv *ImportService) getRequestIdByFileKey(filekey string) (string, error) {
	var q struct {
		workflow.RunTimeConfiguration `graphql:"import_runtime(where: {configuration: {fileKey: {_eq: $fileKey}}, status: {_eq: $status}})"`
	}
	variables := map[string]interface{}{
		"fileKey": graphql.String(filekey),
		"status":  graphql.String("uploading"),
	}
	if err := srv.activities.GraphqlClient.Query(context.Background(), &q, variables); err != nil {
		return "", err
	}
	if len(q.RunTimeConfiguration) == 0 {
		return "", nil
	}
	return string(q.RunTimeConfiguration[0].RequestId), nil
}

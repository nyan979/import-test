package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	"go.temporal.io/sdk/client"
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

	// set requestId to track across go routine
	// app.activities.RequestId = requestId

	// // check if same uploadType is running. reject if true
	// status, err := app.activities.IsAnotherUploadRunning(uploadType, &requestId)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	// if status {
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(http.StatusOK)
	// 	payload := jsonResponse{
	// 		RequestId: requestId,
	// 	}
	// 	jsonPayload, _ := json.Marshal(payload)
	// 	w.Write(jsonPayload)
	// 	return
	// }

	// // insert row into runtime table based on uploadtype configuration
	// config, err := app.activities.ReadConfig(uploadType)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	// err = app.activities.InsertConfigRunTime(requestId, string(config[0].Id))
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	// // minio client call to get presigned url and response back to browser
	// url, err := app.activities.GetPresignedUrl(config)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	//ctx := context.Background()

	// workflowOptions := client.StartWorkflowOptions{
	// 	ID:        requestId,
	// 	TaskQueue: "import-service",
	// }

	status := &workflow.ImportStatus{
		Message: workflow.ImportMessage{
			RequestID: requestId, UploadType: uploadType,
		},
	}

	status, err := CreateImportWorkflow(app.temporalClient, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err = UpdateWorkflow(app.temporalClient, requestId, status)
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

	status, err = UpdateWorkflow(app.temporalClient, requestId, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if status.Stage == "Upload type config not found" {
		http.Error(w, "No such upload type configuration", http.StatusBadRequest)
		return
	}

	status, err = UpdateWorkflow(app.temporalClient, requestId, status)
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

func CreateImportWorkflow(c client.Client, status *workflow.ImportStatus) (*workflow.ImportStatus, error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "import-service",
		ID:        status.Message.RequestID,
	}
	ctx := context.Background()

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workflow.ImportServiceWorkflow)
	if err != nil {
		return status, fmt.Errorf("unable to execute order workflow: %w", err)
	}

	status.Message.RequestID = we.GetID()

	return status, nil
}

func UpdateWorkflow(c client.Client, requestId string, status *workflow.ImportStatus) (*workflow.ImportStatus, error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "import-service",
	}
	ctx := context.Background()

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workflow.SignalImportServiceWorkflow, requestId, status)
	if err != nil {
		return status, fmt.Errorf("unable to execute workflow: %w", err)
	}
	err = we.Get(ctx, &status)
	if err != nil {
		return status, err
	}

	return status, nil
}

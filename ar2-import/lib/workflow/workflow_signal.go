package workflow

import (
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/workflow"
)

const requestSignalName = "request-signal"
const responseSignalName = "response-signal"

type signalRequest struct {
	Status            ImportStatus
	CallingWorkflowId string
}

type signalResponse struct {
	Status ImportStatus
	Error  string
}

func SendErrorResponse(ctx workflow.Context, id string, err error) error {
	logger := workflow.GetLogger(ctx)

	logger.Info("Sending error response", id)

	return workflow.SignalExternalWorkflow(
		ctx,
		id,
		"",
		responseSignalName,
		signalResponse{Error: err.Error()},
	).Get(ctx, nil)
}

func SendResponse(ctx workflow.Context, id string, status ImportStatus) error {
	logger := workflow.GetLogger(ctx)

	logger.Info("Sending response", id)

	return workflow.SignalExternalWorkflow(
		ctx,
		id,
		"",
		responseSignalName,
		signalResponse{Status: status},
	).Get(ctx, nil)
}

func SendRequest(ctx workflow.Context, targetWorkflowID string, status ImportStatus) error {
	logger := workflow.GetLogger(ctx)

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID

	logger.Info("Sending request", targetWorkflowID, workflowID)

	return workflow.SignalExternalWorkflow(
		ctx,
		targetWorkflowID,
		"",
		requestSignalName,
		signalRequest{
			CallingWorkflowId: workflowID,
			Status:            status,
		},
	).Get(ctx, nil)
}

func ReceiveResponse(ctx workflow.Context) (*ImportStatus, error) {
	logger := workflow.GetLogger(ctx)

	var res signalResponse

	ch := workflow.GetSignalChannel(ctx, responseSignalName)

	logger.Info("Waiting for response")

	ch.Receive(ctx, &res)

	logger.Info("Received response")

	if res.Error != "" {
		return nil, fmt.Errorf("%s", res.Error)
	}

	return &res.Status, nil
}

func ReceiveRequest(ctx workflow.Context) (string, *ImportStatus) {
	logger := workflow.GetLogger(ctx)

	var req signalRequest

	ch := workflow.GetSignalChannel(ctx, requestSignalName)

	logger.Info("Waiting for request")

	ch.Receive(ctx, &req)

	logger.Info("Received request")

	return req.CallingWorkflowId, &req.Status
}

func ReceiveRequestWithTimeOut(ctx workflow.Context, timeout time.Duration) (string, *ImportStatus) {
	logger := workflow.GetLogger(ctx)

	var req signalRequest

	ch := workflow.GetSignalChannel(ctx, requestSignalName)

	logger.Info("Waiting for request with timeout")

	ok, more := ch.ReceiveWithTimeout(ctx, time.Second*time.Duration(timeout), &req)
	if !ok && more {
		log.Println("REQUEST TIMEOUT")
		return "", nil
	}

	logger.Info("Received request within timeout duration")

	return req.CallingWorkflowId, &req.Status
}

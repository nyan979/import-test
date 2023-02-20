package workflow

import (
	"fmt"
	"log"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/minio/minio-go/v7"
	"go.temporal.io/sdk/workflow"
)

type Activities struct {
	MinioClient   *minio.Client
	GraphqlClient *graphql.Client
}

type UploadTypeConfiguration []struct {
	Id                        graphql.String
	UploadType                graphql.String
	FileKey                   graphql.String
	UploadExpiryDurationInSec graphql.Int
	DestinationServiceTopic   graphql.String
}

type RunTimeConfiguration []struct {
	RequestId     graphql.String
	ConfigId      graphql.String
	FileVersionId graphql.String
	Status        graphql.String
	Description   graphql.String
	CreatedAt     graphql.String
	UpdatedAt     graphql.String
}

type MinioMessage struct {
	EventName string `json:"EventName"`
	Key       string `json:"Key"`
	Records   []struct {
		EventVersion string `json:"eventVersion"`
		EventSource  string `json:"eventSource"`
		AwsRegion    string `json:"awsRegion"`
		EventTime    string `json:"eventTime"`
		EventName    string `json:"eventName"`
		UserIdentity struct {
			PrincipalId string `json:"principalId"`
		} `json:"userIdentity"`
		RequestParameters struct {
			PrincipalId     string `json:"principalId"`
			Region          string `json:"region"`
			SourceIPAddress string `json:"sourceIPAddress"`
		} `json:"requestParameters"`
		ResponseElements struct {
			Content_Length          string `json:"content-length"`
			X_AMZ_RequestId         string `json:"x-amz-request-id"`
			X_MINIO_Deployment_Id   string `json:"x-minio-deployment-id"`
			X_MINIO_Origin_Endpoint string `json:"x-minio-origin-endpoint"`
		} `json:"responseElements"`
		S3 struct {
			S3SchemaVersion string `json:"s3SchemaVersion"`
			ConfigurationId string `json:"configurationId"`
			Bucket          struct {
				Name          string `json:"name"`
				OwnerIdentity struct {
					PrincipalId string `json:"principalId"`
				}
				ARN string `json:"arn"`
			} `json:"bucket"`
			Object struct {
				Key          string `json:"key"`
				Size         int    `json:"size"`
				ETag         string `json:"eTag"`
				ContentType  string `json:"contentType"`
				UserMetaData struct {
					ContentType                         string `json:"contentType"`
					X_AMZ_Object_Lock_Mode              string `json:"x-amz-object-lock-mode"`
					X_AMZ_Object_Lock_Retain_Until_Date string `json:"x-amz-object-lock-retain-until-date"`
				} `json:"userMetadata"`
				VersionId string `json:"versionId"`
				Sequencer string `json:"sequencer"`
			} `json:"Object"`
		} `json:"s3"`
		Source struct {
			Host      string `json:"host"`
			Port      string `json:"port"`
			UserAgent string `json:"userAgent"`
		} `json:"source"`
	} `json:"Records"`
}

type ImportMessage struct {
	RequestID     string
	UploadType    string
	FileKey       string
	FileVersionID string
	URL           string
	Record        []string
}

const (
	ServiceBusyStage  = "ServiceBusy"
	UploadTypeStage   = "UploadType"
	PresignedUrlStage = "PresignedUrl"
	ParseCsvStage     = "ParseCsv"
)

type ImportSignal struct {
	Message ImportMessage
	Stage   string
}

const requestSignalName = "request-signal"
const responseSignalName = "response-signal"

type signalRequest struct {
	Signal            ImportSignal
	CallingWorkflowId string
}

type signalResponse struct {
	Signal ImportSignal
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

func SendResponse(ctx workflow.Context, id string, signal ImportSignal) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Sending response", id)
	return workflow.SignalExternalWorkflow(
		ctx,
		id,
		"",
		responseSignalName,
		signalResponse{Signal: signal},
	).Get(ctx, nil)
}

func SendRequest(ctx workflow.Context, targetWorkflowID string, signal ImportSignal) error {
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
			Signal:            signal,
		},
	).Get(ctx, nil)
}

func ReceiveResponse(ctx workflow.Context) (*ImportSignal, error) {
	logger := workflow.GetLogger(ctx)
	var res signalResponse
	ch := workflow.GetSignalChannel(ctx, responseSignalName)
	logger.Info("Waiting for response")
	ch.Receive(ctx, &res)
	logger.Info("Received response")

	if res.Error != "" {
		return nil, fmt.Errorf("%s", res.Error)
	}
	return &res.Signal, nil
}

func ReceiveRequest(ctx workflow.Context) (string, *ImportSignal) {
	logger := workflow.GetLogger(ctx)
	var req signalRequest
	ch := workflow.GetSignalChannel(ctx, requestSignalName)
	logger.Info("Waiting for request")
	ch.Receive(ctx, &req)
	logger.Info("Received request")
	return req.CallingWorkflowId, &req.Signal
}

func ReceiveRequestWithTimeOut(ctx workflow.Context, timeout time.Duration) (string, *ImportSignal) {
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
	return req.CallingWorkflowId, &req.Signal
}

package workflow

import (
	"log"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ImportMessage struct {
	RequestID     string
	UploadType    string
	FileKey       string
	FileVersionID string
	URL           string
	Record        []string
}

type ImportStatus struct {
	Message ImportMessage
	Stage   string
}

func ImportServiceWorkflow(ctx workflow.Context) error {
	retrypolicy := &temporal.RetryPolicy{
		MaximumAttempts: 1,
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy:         retrypolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Println("Start of workflow")

	workflowId, status := ReceiveRequest(ctx)
	var newRequestId string
	var config UploadTypeConfiguration

	log.Println(status)

	log.Println("Execute IsServiceBusy")

	err := workflow.ExecuteActivity(ctx, activities.IsServiceBusy, status.Message.UploadType).Get(ctx, &newRequestId)
	if err != nil {
		return err
	}

	if len(newRequestId) > 0 {
		status.Message.RequestID = newRequestId
		status.Stage = "Service Busy"
		err = SendResponse(ctx, workflowId, *status)
		if err != nil {
			return err
		}
		return nil
	}

	log.Println("Execute ReadConfig")

	err = workflow.ExecuteActivity(ctx, activities.ReadConfig, status.Message.UploadType).Get(ctx, &config)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	log.Println(config)

	if config == nil {
		status.Stage = "Upload type not found"
		err = SendResponse(ctx, workflowId, *status)
		if err != nil {
			return err
		}
		return nil
	}

	err = workflow.ExecuteActivity(ctx, activities.GetPresignedUrl, config).Get(ctx, &status.Message.URL)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	status.Stage = "Presigned Url"
	err = SendResponse(ctx, workflowId, *status)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, activities.InsertConfigRunTime, status.Message.RequestID, config[0].Id).Get(ctx, nil)
	if err != nil {
		return err
	}

	var statusNew *ImportStatus

	_, statusNew = ReceiveRequestWithTimeOut(ctx, time.Duration(config[0].UploadExpiryDurationInSec))
	if statusNew == nil {
		err = workflow.ExecuteActivity(ctx, activities.UpdateConfigRunTimeStatus, status.Message.RequestID, "failed").Get(ctx, nil)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
		}
		return nil
	}

	err = workflow.ExecuteActivity(ctx, activities.UpdateConfigRunTimeStatus, status.Message.RequestID, "importing").Get(ctx, nil)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	err = workflow.ExecuteActivity(ctx, activities.ParseCSV, status.Message.FileKey).Get(ctx, &status.Message.Record)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	status.Stage = "Parsed CSV content"
	err = SendResponse(ctx, workflowId, *status)
	if err != nil {
		return err
	}

	log.Println("End of workflow")

	return nil
}

func SignalImportServiceWorkflow(ctx workflow.Context, orderWorkflowID string, status *ImportStatus) (*ImportStatus, error) {
	log.Println("Send Request...")
	err := SendRequest(ctx, orderWorkflowID, *status)
	if err != nil {
		return status, err
	}

	log.Println("Receive Response...")
	status, err = ReceiveResponse(ctx)
	if err != nil {
		return status, err
	}

	return status, nil
}

package workflow

import (
	"log"
	"time"

	"go.temporal.io/sdk/workflow"
)

type ImportMessage struct {
	RequestID  string
	UploadType string
	Config     UploadTypeConfiguration
	RunConfig  RunTimeConfiguration
	URL        string
}

type ImportStatus struct {
	Message ImportMessage
	Stage   string
}

func ImportServiceWorkflow(ctx workflow.Context) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var status *ImportStatus
	var workflowId string

	for {
		workflowId, status = ReceiveRequest(ctx)
		var newRequestId string

		err := workflow.ExecuteActivity(ctx, activities.IsAnotherUploadRunning, status.Message.UploadType).Get(ctx, &newRequestId)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
			continue
		}

		if len(newRequestId) > 0 {
			status.Message.RequestID = newRequestId
			status.Stage = "Service not available"
			err = SendResponse(ctx, workflowId, *status)
			if err != nil {
				err := SendErrorResponse(ctx, workflowId, err)
				if err != nil {
					return err
				}
			}
			return nil
		}
		err = SendResponse(ctx, workflowId, *status)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
		}
		break
	}

	for {
		workflowId, status = ReceiveRequest(ctx)

		err := workflow.ExecuteActivity(ctx, activities.ReadConfig, status.Message.UploadType).Get(ctx, &status.Message.Config)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
			continue
		}
		if status.Message.Config == nil {
			status.Stage = "Upload type config not found"
			err = SendResponse(ctx, workflowId, *status)
			if err != nil {
				err := SendErrorResponse(ctx, workflowId, err)
				if err != nil {
					return err
				}
			}
			return nil
		}
		err = SendResponse(ctx, workflowId, *status)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
		}
		break
	}

	for {
		workflowId, status = ReceiveRequest(ctx)

		err := workflow.ExecuteActivity(ctx, activities.GetPresignedUrl, status.Message.Config).Get(ctx, &status.Message.URL)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
			continue
		}
		if len(status.Message.URL) == 0 {
			err = SendResponse(ctx, workflowId, *status)
			if err != nil {
				err := SendErrorResponse(ctx, workflowId, err)
				if err != nil {
					return err
				}
			}
			return nil
		}
		status.Stage = "Presigned Url"
		err = SendResponse(ctx, workflowId, *status)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
		}
		break
	}

	err := workflow.ExecuteActivity(ctx, activities.InsertConfigRunTime, status.Message.RequestID, status.Message.Config[0].Id).Get(ctx, nil)
	if err != nil {
		return err
	}

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

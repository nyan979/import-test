package workflow

import (
	"log"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ImportMessage struct {
	RequestID  string
	UploadType string
	FileKey    string
	Config     UploadTypeConfiguration
	RunConfig  RunTimeConfiguration
	URL        string
	Record     []string
}

type ImportStatus struct {
	Message ImportMessage
	Stage   string
}

func ImportServiceWorkflow(ctx workflow.Context) error {
	retrypolicy := &temporal.RetryPolicy{
		// InitialInterval:        time.Second,
		// BackoffCoefficient:     2.0,
		// MaximumInterval:        time.Second * 100,
		MaximumAttempts: 1,
		// NonRetryableErrorTypes: []string{},
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy:         retrypolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Println("Start Workflow 1")

	workflowId, status := ReceiveRequest(ctx)
	var newRequestId string

	err := workflow.ExecuteActivity(ctx, activities.IsAnotherUploadRunning, status.Message.UploadType).Get(ctx, &newRequestId)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
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

	err = workflow.ExecuteActivity(ctx, activities.ReadConfig, status.Message.UploadType).Get(ctx, &status.Message.Config)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
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

	err = workflow.ExecuteActivity(ctx, activities.GetPresignedUrl, status.Message.Config).Get(ctx, &status.Message.URL)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	status.Stage = "Presigned Url"

	err = SendResponse(ctx, workflowId, *status)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	err = workflow.ExecuteActivity(ctx, activities.InsertConfigRunTime, status.Message.RequestID, status.Message.Config[0].Id).Get(ctx, nil)
	if err != nil {
		return err
	}

	log.Println("Start Workflow 2")

	cwo := workflow.ChildWorkflowOptions{
		WorkflowID:         status.Message.RequestID + "-child",
		WorkflowRunTimeout: 10 * time.Second,
	}

	ctx = workflow.WithChildOptions(ctx, cwo)

	future := workflow.ExecuteChildWorkflow(ctx, WaitForMinioNotification, status)

	log.Println(future)

	err = workflow.ExecuteActivity(ctx, activities.UpdateConfigRunTimeFileVersion, status.Message.RunConfig).Get(ctx, nil)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	log.Println("Outside Session")

	err = workflow.ExecuteActivity(ctx, activities.UpdateConfigRunTimeStatus, "Importing").Get(ctx, nil)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	err = workflow.ExecuteActivity(ctx, activities.ParseCSVToLine, status.Message.FileKey).Get(ctx, &status.Message.Record)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	status.Stage = "Parsed CSV content"
	err = SendResponse(ctx, workflowId, *status)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}

	log.Println("End of workflow")

	return nil
}

func WaitForMinioNotification(ctx workflow.Context) (*ImportStatus, error) {
	_, status := ReceiveRequest(ctx)

	log.Println("Inside Session")

	return status, nil
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

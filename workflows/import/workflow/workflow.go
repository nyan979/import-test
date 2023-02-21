package workflow

import (
	"log"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var activities Activities

func ImportServiceWorkflow(ctx workflow.Context) error {
	retrypolicy := &temporal.RetryPolicy{
		MaximumAttempts: 1,
	}
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy:         retrypolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	workflowId, signal := ReceiveRequest(ctx)
	var newRequestId string
	var config UploadTypeConfiguration
	err := workflow.ExecuteActivity(ctx, activities.GetBusyRuntimeRequestId, signal.Message.UploadType).Get(ctx, &newRequestId)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}
	if len(newRequestId) > 0 {
		signal.Message.RequestID = newRequestId
		signal.Stage = ServiceBusyStage
		err = SendResponse(ctx, workflowId, signal)
		if err != nil {
			return err
		}
		return nil
	}
	err = workflow.ExecuteActivity(ctx, activities.GetConfiguration, signal.Message.UploadType).Get(ctx, &config)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}
	if config == nil {
		signal.Stage = UploadTypeStage
		err = SendResponse(ctx, workflowId, signal)
		if err != nil {
			return err
		}
		return nil
	}
	err = workflow.ExecuteActivity(ctx, activities.GetPresignedUrl, config).Get(ctx, &signal.Message.URL)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}
	signal.Stage = PresignedUrlStage
	err = SendResponse(ctx, workflowId, signal)
	if err != nil {
		return err
	}
	err = workflow.ExecuteActivity(ctx, activities.InsertConfigRunTime, signal.Message.RequestID, config[0].Id).Get(ctx, nil)
	if err != nil {
		return err
	}
	var signalNew *ImportSignal
	workflowId, signalNew = ReceiveRequestWithTimeOut(ctx, time.Duration(config[0].UploadExpiryDurationInSec))
	if signalNew == nil {
		err = workflow.ExecuteActivity(ctx, activities.UpdateConfigRunTimeStatus, signal.Message.RequestID, "failed").Get(ctx, nil)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
		}
		return nil
	}
	signal = signalNew
	err = workflow.ExecuteActivity(ctx, activities.UpdateConfigRunTimeStatus, signal.Message.RequestID, "importing").Get(ctx, nil)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}
	err = workflow.ExecuteActivity(ctx, activities.ParseCSV, signal.Message.FileKey).Get(ctx, &signal.Message.Record)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}
	signal.Stage = ParseCsvStage
	err = SendResponse(ctx, workflowId, signal)
	if err != nil {
		return err
	}
	return nil
}

func SignalImportServiceWorkflow(ctx workflow.Context, orderWorkflowID string, signal *ImportSignal) (*ImportSignal, error) {
	log.Println("Send Request...")
	err := SendRequest(ctx, orderWorkflowID, signal)
	if err != nil {
		return signal, err
	}
	log.Println("Receive Response...")
	signal, err = ReceiveResponse(ctx)
	if err != nil {
		return signal, err
	}
	return signal, nil
}

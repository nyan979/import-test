package workflow

import (
	"log"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var activities Activities

func ImportServiceWorkflow(ctx workflow.Context) error {
	log.Println("Starting new workflow...")
	retrypolicy := &temporal.RetryPolicy{
		MaximumAttempts: 1,
	}
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy:         retrypolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	workflowId, signal := ReceiveRequest(ctx)
	log.Println("Recieve request: ", workflowId, signal)
	var newRequestId string
	var config UploadTypeConfiguration
	log.Println("Executing GetBusyRuntimeRequestId...")
	err := workflow.ExecuteActivity(ctx, activities.GetBusyRuntimeRequestId, signal.Message.UploadType).Get(ctx, &newRequestId)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}
	log.Println("New requestId: ", newRequestId)
	if len(newRequestId) > 0 {
		signal.Message.RequestID = newRequestId
		signal.Stage = ServiceBusyStage
		log.Println("Sending signal: ", ServiceBusyStage)
		err = SendResponse(ctx, workflowId, signal)
		if err != nil {
			return err
		}
		return nil
	}
	log.Println("Executing GetConfiguration...")
	err = workflow.ExecuteActivity(ctx, activities.GetConfiguration, signal.Message.UploadType).Get(ctx, &config)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}
	log.Println("Config: ", config)
	if config == nil {
		signal.Stage = UploadTypeStage
		log.Println("Sending signal: ", UploadTypeStage)
		err = SendResponse(ctx, workflowId, signal)
		if err != nil {
			return err
		}
		return nil
	}
	log.Println("Executing GetPresignedUrl...")
	err = workflow.ExecuteActivity(ctx, activities.GetPresignedUrl, config).Get(ctx, &signal.Message.URL)
	if err != nil {
		err := SendErrorResponse(ctx, workflowId, err)
		if err != nil {
			return err
		}
	}
	signal.Stage = PresignedUrlStage
	log.Println("Sending signal: ", PresignedUrlStage)
	err = SendResponse(ctx, workflowId, signal)
	if err != nil {
		return err
	}
	log.Println("Executing InsertConfigRunTime...")
	err = workflow.ExecuteActivity(ctx, activities.InsertConfigRunTime, signal.Message.RequestID, config[0].Id).Get(ctx, nil)
	if err != nil {
		return err
	}
	var signalNew *ImportSignal
	log.Println("Waiting for notification in seconds", time.Duration(config[0].UploadExpiryDurationInSec))
	workflowId, signalNew = ReceiveRequestWithTimeOut(ctx, time.Duration(config[0].UploadExpiryDurationInSec))
	if signalNew == nil {
		log.Println("Executing UpdateConfigRunTimeStatus failed...")
		err = workflow.ExecuteActivity(ctx, activities.UpdateConfigRunTimeStatus, signal.Message.RequestID, "failed").Get(ctx, nil)
		if err != nil {
			err := SendErrorResponse(ctx, workflowId, err)
			if err != nil {
				return err
			}
		}
		log.Println("Exiting")
		return nil
	}
	signal = signalNew
	log.Println("Executing UpdateConfigRunTimeStatus importing...")
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

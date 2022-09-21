package workflow

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func CSVImportServiceWorkflow(ctx workflow.Context, uploadType string) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	config := UploadTypeConfiguration{}

	err := workflow.ExecuteActivity(ctx, activities.ReadConfig, uploadType).Get(ctx, &config)
	if err != nil {
		return err
	}

	var requestId string

	err = workflow.ExecuteActivity(ctx, activities.GetPresignedUrl, config).Get(ctx, &requestId)
	if err != nil {
		return err
	}

	return nil
}

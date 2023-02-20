package workflow_test

import (
	"mssfoobar/ar2-import/workflows/import/workflow"
	"testing"

	"github.com/hasura/go-graphql-client"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestImportServiceWorkflow(t *testing.T) {
	s := testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(workflow.ImportServiceWorkflow)
	env.RegisterWorkflow(workflow.SignalImportServiceWorkflow)

	env.RegisterActivity(workflow.Activities{
		MinioClient:   &minio.Client{},
		GraphqlClient: &graphql.Client{},
	})

	requestId := "testRequestId"
	signal := workflow.ImportSignal{
		Message: workflow.ImportMessage{
			RequestID:     requestId,
			UploadType:    "testUploadType",
			FileKey:       "testFileKey",
			FileVersionID: "testFileVersionID",
			URL:           "http://testPresignedUrl",
			Record: []string{
				"testRecord1",
				"testRecord2",
			},
		},
	}

	env.ExecuteWorkflow(workflow.ImportServiceWorkflow)
	env.ExecuteWorkflow(workflow.SignalImportServiceWorkflow, requestId, signal)
	env.ExecuteWorkflow(workflow.SignalImportServiceWorkflow, requestId, signal)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

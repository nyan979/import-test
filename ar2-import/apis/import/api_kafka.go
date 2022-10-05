package main

import (
	"context"
	"encoding/json"
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"

	"github.com/hasura/go-graphql-client"
	"github.com/segmentio/kafka-go"
)

type jsonMessage struct {
	RequestId string `json:"requestId"`
	Record    string `json:"record"`
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

func (app *Application) FetchMinioMessage(ctx context.Context, messages chan<- kafka.Message) error {
	log.Println("Kafka Fetch Message Starting...")
	for {
		msg, err := app.KafkaMinioReader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case messages <- msg:
			log.Printf("Fetched an msg: %v \n", string(msg.Value))
		}
	}
}

func (app *Application) FetchImportMessage(ctx context.Context, messages chan<- kafka.Message, messagesCommit chan<- kafka.Message) error {
	log.Println("Kafka Fetch Message Starting...")
	for {
		msg, err := app.KafkaImportReader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case messages <- msg:
			log.Printf("Fetched an msg: %v \n", string(msg.Value))

			err = app.updateConfigRunTimeStatus(string(msg.Value), "done")
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
			case messagesCommit <- msg:
			}
		}
	}
}

func (app *Application) WriteMessages(ctx context.Context, messages <-chan kafka.Message, messagesCommit chan<- kafka.Message) error {
	log.Println("Kafka Write Message Starting...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-messages:
			var minioMsg MinioMessage

			err := json.Unmarshal(msg.Value, &minioMsg)
			if err != nil {
				return err
			}

			status := &workflow.ImportStatus{}
			status.Message.FileKey = minioMsg.Records[0].S3.Object.Key
			status.Message.FileVersionID = minioMsg.Records[0].S3.Object.VersionId

			status.Message.RequestID, err = app.getRequestIDFromRunConfig(status.Message.FileKey)
			if err != nil {
				return err
			}

			status, err = utils.ExecuteImportWorkflow(app.temporalClient, string(status.Message.RequestID), status)
			if err != nil {
				return err
			}

			var jsonMessages []jsonMessage

			for _, line := range status.Message.Record {
				jsonMessages = append(jsonMessages, jsonMessage{
					RequestId: string(status.Message.RequestID),
					Record:    line,
				})
			}

			var kafkaMessage []kafka.Message
			for _, msg := range jsonMessages {
				jsonByte, _ := json.Marshal(msg)
				kafkaMessage = append(kafkaMessage, kafka.Message{
					Value: jsonByte,
				})
			}

			err = app.KafkaWriter.WriteMessages(ctx, kafkaMessage...)
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
			case messagesCommit <- msg:
			}
		}
	}
}

func (app *Application) CommitMinioMessages(ctx context.Context, messagesCommit <-chan kafka.Message) error {
	for {
		select {
		case <-ctx.Done():
		case msg := <-messagesCommit:
			err := app.KafkaMinioReader.CommitMessages(ctx, msg)
			if err != nil {
				return err
			}
			log.Printf("committed an msg: %v \n", string(msg.Value))
		}
	}
}

func (app *Application) CommitImportMessages(ctx context.Context, messagesCommit <-chan kafka.Message) error {
	for {
		select {
		case <-ctx.Done():
		case msg := <-messagesCommit:
			err := app.KafkaImportReader.CommitMessages(ctx, msg)
			if err != nil {
				return err
			}
			log.Printf("committed an msg: %v \n", string(msg.Value))
		}
	}
}

func (app *Application) getRequestIDFromRunConfig(filename string) (string, error) {
	var q1 struct {
		workflow.RunTimeConfiguration `graphql:"import_runtime(where: {configuration: {fileKey: {_eq: $fileKey}}, status: {_eq: $status}})"`
	}

	variables := map[string]interface{}{
		"fileKey": graphql.String(filename),
		"status":  graphql.String("uploading"),
	}

	if err := app.activities.GraphqlClient.Query(context.Background(), &q1, variables); err != nil {
		log.Fatal(err)
		return "", err
	}

	if len(q1.RunTimeConfiguration) == 0 {
		return "", nil
	}

	return string(q1.RunTimeConfiguration[0].RequestId), nil
}

func (app *Application) updateConfigRunTimeStatus(requestId string, status string) error {
	var mutation struct {
		UpdateData struct {
			UpdatedAt string
		} `graphql:"update_import_runtime_by_pk(pk_columns: {requestId: $rqId}, _set: {status: $status})"`
	}

	variables := map[string]interface{}{
		"rqId":   graphql.String(requestId),
		"status": graphql.String(status),
	}

	log.Println(variables)

	if err := app.activities.GraphqlClient.Mutate(context.Background(), &mutation, variables); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

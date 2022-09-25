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

func (app *Application) FetchMessage(ctx context.Context, messages chan<- kafka.Message) error {
	log.Println("Kafka Fetch Message Starting...")
	for {
		msg, err := app.KafkaReader.FetchMessage(ctx)
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

			filename := minioMsg.Records[0].S3.Object.Key

			status := &workflow.ImportStatus{}

			status.Message.FileKey = filename

			status.Message.RunConfig, err = app.activities.GetRuntimeConfig(filename)
			if err != nil {
				return err
			}

			status.Message.RunConfig[0].FileVersionId = graphql.String(minioMsg.Records[0].S3.Object.VersionId)

			status, err = utils.UpdateWorkflow(app.temporalClient, string(status.Message.RunConfig[0].RequestId), status)
			if err != nil {
				return err
			}

			var jsonMessages []jsonMessage

			for _, line := range status.Message.Record {
				jsonMessages = append(jsonMessages, jsonMessage{
					RequestId: string(status.Message.RunConfig[0].RequestId),
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

func (app *Application) CommitMessages(ctx context.Context, messagesCommit <-chan kafka.Message) error {
	for {
		select {
		case <-ctx.Done():
		case msg := <-messagesCommit:
			err := app.KafkaReader.CommitMessages(ctx, msg)
			if err != nil {
				return err
			}
			log.Printf("committed an msg: %v \n", string(msg.Value))
		}
	}
}

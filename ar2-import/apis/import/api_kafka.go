package main

import (
	"context"
	"encoding/json"
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"
	"os"

	"github.com/segmentio/kafka-go"
)

type Message struct {
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

func (app *Application) StartKafka() {
	conf := kafka.ReaderConfig{
		Brokers:  []string{os.Getenv("KAFKA_HOST") + ":" + os.Getenv("KAFKA_PORT")},
		Topic:    os.Getenv("KAFKA_IMPORT_TOPIC"),
		GroupID:  "1",
		MaxBytes: 10,
	}

	r := kafka.NewReader(conf)

	var msg Message

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Fatalln(err)
			continue
		}
		err = json.Unmarshal(m.Value, &msg)
		if err != nil {
			log.Fatalln(err)
			continue
		}
		log.Println(msg.Records[0].S3.Object.VersionId)
		log.Println(msg.Records[0].S3.Object.Key)

		var activities workflow.Activities
		activities.GraphQlClient = app.graphqlClient
		activities.MinioClient = app.minioClient

		activities.GetObject(msg.Records[0].S3.Object.Key)
	}
}

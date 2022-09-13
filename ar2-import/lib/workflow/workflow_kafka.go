package workflow

import (
	"context"
	"encoding/json"
	"log"

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

func (a *Activities) ReadMessage(ctx context.Context, messages chan<- kafka.Message) error {
	var msg Message

	for {
		message, err := a.KafkaReader.ReadMessage(ctx)
		if err != nil {
			return err
		}

		err = json.Unmarshal(message.Value, &msg)
		if err != nil {
			log.Fatalln(err)
			continue
		}
		log.Println(msg.Records[0].S3.Object.VersionId)
		log.Println(msg.Records[0].S3.Object.Key)

		lines, err := a.ParseLine(msg.Records[0].S3.Object.Key)
		if err != nil {
			log.Fatalln(err)
			continue
		}

		for _, v := range lines {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case messages <- kafka.Message{
				Value: []byte(v),
			}:
				log.Printf("message fetched and sent to a channel: %v \n", string(message.Value))

			}
		}
	}
}

func (a *Activities) CommitMessages(ctx context.Context, messageCommitChan <-chan kafka.Message) error {
	for {
		select {
		case <-ctx.Done():
		case msg := <-messageCommitChan:
			err := a.KafkaReader.CommitMessages(ctx, msg)
			if err != nil {
				return err
			}
			log.Printf("committed an msg: %v \n", string(msg.Value))
		}
	}
}

func (a *Activities) WriteMessages(ctx context.Context, messages chan kafka.Message, messageCommitChan chan kafka.Message) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case m := <-messages:
			err := a.KafkaWriter.WriteMessages(ctx, kafka.Message{
				Value: m.Value,
			})
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
			case messageCommitChan <- m:
			}
		}
	}
}

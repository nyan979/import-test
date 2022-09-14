package workflow

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/minio/minio-go"
	"github.com/segmentio/kafka-go"
)

type Activities struct {
	MinioClient   *minio.Client
	GraphQlClient *graphql.Client
	KafkaReader   *kafka.Reader
	KafkaWriter   *kafka.Writer
}

type UploadTypeConfiguration []struct {
	Id                        graphql.String
	UploadType                graphql.String
	FileKey                   graphql.String
	UploadExpiryDurationInSec graphql.Int
	DestinationServiceTopic   graphql.String
}

type RunTimeConfiguration []struct {
	RequestId     graphql.String
	ConfigId      graphql.String
	FileVersionId graphql.String
	Status        graphql.String
	Description   graphql.String
	CreatedAt     graphql.String
	UpdatedAt     graphql.String
}

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

func (a *Activities) ReadConfigTable(uploadType string) (UploadTypeConfiguration, error) {
	var q struct {
		UploadTypeConfiguration `graphql:"import_configuration(where: {uploadType: {_eq: $configUploadType}})"`
	}

	variables := map[string]interface{}{
		"configUploadType": graphql.String(uploadType),
	}

	if err := a.GraphQlClient.Query(context.Background(), &q, variables); err != nil {
		log.Fatal(err)
		return nil, err
	}

	if len(q.UploadTypeConfiguration) == 0 {
		s := "There is no " + uploadType + " configuration."
		return nil, errors.New(s)
	}

	return q.UploadTypeConfiguration, nil
}

func (a *Activities) GetPresignedUrl(config UploadTypeConfiguration) (*url.URL, error) {
	presignedURL, err := a.MinioClient.PresignedPutObject(os.Getenv("MINIO_BUCKET_NAME"), string(config[0].FileKey)+".csv",
		time.Duration(config[0].UploadExpiryDurationInSec)*time.Second)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	return presignedURL, nil
}

func (a *Activities) IsRequestIdBusy(requestId *string) (bool, error) {
	var q struct {
		RunTimeConfiguration `graphql:"import_runtime(where: {requestId: {_eq: $reqId}})"`
	}

	variables := map[string]interface{}{
		"reqId": graphql.String(*requestId),
	}

	if err := a.GraphQlClient.Query(context.Background(), &q, variables); err != nil {
		return false, err
	}

	if len(q.RunTimeConfiguration) == 0 {
		return false, nil
	}

	switch status := q.RunTimeConfiguration[0].Status; status {
	case "uploading":
		*requestId = string(q.RunTimeConfiguration[0].RequestId)
		return true, nil
	case "importing":
		*requestId = string(q.RunTimeConfiguration[0].RequestId)
		return true, nil
	case "completed":
		return false, nil
	case "failed":
		return false, nil
	default:
		return false, nil
	}
}

func (a *Activities) InsertConfigRunTimeTable(requestId string, configId string) error {
	type import_runtime_insert_input struct {
		RequestId   string `json:"requestId"`
		ConfigId    string `json:"configId"`
		Status      string `json:"status"`
		Description string `json:"description,omitempty"`
	}

	var mutation struct {
		InsertData struct {
			CreatedAt graphql.String
			UpdatedAt graphql.String
		} `graphql:"insert_import_runtime_one(object: $object)"`
	}

	variables := map[string]interface{}{
		"object": import_runtime_insert_input{
			RequestId:   requestId,
			ConfigId:    configId,
			Status:      "uploading",
			Description: "testing",
		},
	}

	if err := a.GraphQlClient.Mutate(context.Background(), &mutation, variables); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func (a *Activities) ReadMinioNotification(message kafka.Message) (string, error) {
	var msg Message

	err := json.Unmarshal(message.Value, &msg)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	return msg.Records[0].S3.Object.Key, nil
}

func (a *Activities) ParseCSVToLine(filekey string) ([]string, error) {
	object, err := a.MinioClient.GetObject(os.Getenv("MINIO_BUCKET_NAME"), filekey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	var lines []string

	scanner := bufio.NewScanner(bufio.NewReader(object))

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, nil
}

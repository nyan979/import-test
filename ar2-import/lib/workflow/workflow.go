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
	"github.com/minio/minio-go/v7"
	"github.com/segmentio/kafka-go"
)

type Activities struct {
	MinioClient   *minio.Client
	GraphqlClient *graphql.Client
	KafkaReader   *kafka.Reader
	KafkaWriter   *kafka.Writer
	RequestId     string
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

func (a *Activities) ReadConfig(uploadType string) (UploadTypeConfiguration, error) {
	var q struct {
		UploadTypeConfiguration `graphql:"import_configuration(where: {uploadType: {_eq: $configUploadType}})"`
	}

	variables := map[string]interface{}{
		"configUploadType": graphql.String(uploadType),
	}

	if err := a.GraphqlClient.Query(context.Background(), &q, variables); err != nil {
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
	presignedURL, err := a.MinioClient.PresignedPutObject(context.Background(), os.Getenv("MINIO_BUCKET_NAME"), string(config[0].FileKey)+".csv",
		time.Duration(config[0].UploadExpiryDurationInSec)*time.Second)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	return presignedURL, nil
}

func (a *Activities) IsAnotherUploadRunning(uploadType string, requestId *string) (bool, error) {
	var q struct {
		RunTimeConfiguration `graphql:"import_runtime(where: {configuration: {uploadType: {_eq: $uploadType}}}, distinct_on: status)"`
	}

	variables := map[string]interface{}{
		"uploadType": graphql.String(uploadType),
	}

	if err := a.GraphqlClient.Query(context.Background(), &q, variables); err != nil {
		return false, err
	}

	if len(q.RunTimeConfiguration) == 0 {
		return false, nil
	}

	for _, v := range q.RunTimeConfiguration {
		if v.Status == "uploading" || v.Status == "importing" {
			*requestId = string(v.RequestId)
			return true, nil
		}
	}

	return false, nil
}

func (a *Activities) InsertConfigRunTime(requestId string, configId string) error {
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

	if err := a.GraphqlClient.Mutate(context.Background(), &mutation, variables); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func (a *Activities) UpdateConfigRunTimeFileVersion(requestId string, version string) error {
	var mutation struct {
		UpdateData struct {
			UpdatedAt graphql.String
		} `graphql:"update_import_runtime_by_pk(pk_columns: {requestId: $rqId}, _set: {fileVersionId: $verId})"`
	}

	variables := map[string]interface{}{
		"rqId":  graphql.String(requestId),
		"verId": graphql.String(version),
	}

	if err := a.GraphqlClient.Mutate(context.Background(), &mutation, variables); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func (a *Activities) UpdateConfigRunTimeStatus(requestId string, status string) error {
	var mutation struct {
		UpdateData struct {
			UpdatedAt graphql.String
		} `graphql:"update_import_runtime_by_pk(pk_columns: {requestId: $rqId}, _set: {status: $status})"`
	}

	variables := map[string]interface{}{
		"rqId":   graphql.String(requestId),
		"status": graphql.String(status),
	}

	if err := a.GraphqlClient.Mutate(context.Background(), &mutation, variables); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func (a *Activities) ReadMinioNotification(message kafka.Message) (*MinioMessage, error) {
	var msg MinioMessage

	err := json.Unmarshal(message.Value, &msg)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	return &msg, nil
}

func (a *Activities) ParseCSVToLine(filekey string) ([]string, error) {
	object, err := a.MinioClient.GetObject(context.Background(), os.Getenv("MINIO_BUCKET_NAME"), filekey, minio.GetObjectOptions{})
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

package workflow

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/minio/minio-go"
)

type Activities struct {
	MinioClient   *minio.Client
	GraphQlClient *graphql.Client
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

func (a *Activities) ReadConfigTable(uploadType string) (UploadTypeConfiguration, error) {
	var q struct {
		UploadTypeConfiguration `graphql:"import_config_config(where: {uploadType: {_eq: $configUploadType}})"`
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
	presignedURL, err := a.MinioClient.PresignedPutObject(os.Getenv("MINIO_BUCKET_NAME"), string(config[0].FileKey),
		time.Duration(config[0].UploadExpiryDurationInSec)*time.Second)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	return presignedURL, nil
}

func (a *Activities) IsRequestIdBusy(requestId *string) (bool, error) {
	var q struct {
		RunTimeConfiguration `graphql:"import_config_runtime(where: {requestId: {_eq: $reqId}})"`
	}

	log.Println(*requestId)

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
	type import_config_runtime_insert_input struct {
		RequestId   string `json:"requestId"`
		ConfigId    string `json:"configId"`
		Status      string `json:"status"`
		Description string `json:"description,omitempty"`
	}

	var mutation struct {
		InsertData struct {
			CreatedAt graphql.String
			UpdatedAt graphql.String
		} `graphql:"insert_import_config_runtime_one(object: $object)"`
	}

	variables := map[string]interface{}{
		"object": import_config_runtime_insert_input{
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

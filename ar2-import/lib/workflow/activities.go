package workflow

import (
	"bufio"
	"context"
	"log"
	"os"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/minio/minio-go"
)

func (a *Activities) GetConfiguration(uploadType string) (UploadTypeConfiguration, error) {
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
		return nil, nil
	}

	return q.UploadTypeConfiguration, nil
}

func (a *Activities) GetPresignedUrl(config UploadTypeConfiguration) (string, error) {
	presignedURL, err := a.MinioClient.PresignedPutObject(os.Getenv("MINIO_BUCKET_NAME"), string(config[0].FileKey),
		time.Duration(config[0].UploadExpiryDurationInSec)*time.Second)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	return presignedURL.String(), nil
}

func (a *Activities) GetBusyRuntimeRequestId(uploadType string) (string, error) {
	var q struct {
		RunTimeConfiguration `graphql:"import_runtime(where: {configuration: {uploadType: {_eq: $uploadType}}}, distinct_on: status)"`
	}

	variables := map[string]interface{}{
		"uploadType": graphql.String(uploadType),
	}

	if err := a.GraphqlClient.Query(context.Background(), &q, variables); err != nil {
		return "", err
	}

	for _, v := range q.RunTimeConfiguration {
		if v.Status == "uploading" || v.Status == "importing" {
			return string(v.RequestId), nil
		}
	}

	return "", nil
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
			CreatedAt string
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

func (a *Activities) UpdateConfigRunTimeFileVersion(runConfig RunTimeConfiguration) error {
	var mutation struct {
		UpdateData struct {
			UpdatedAt string
		} `graphql:"update_import_runtime_by_pk(pk_columns: {requestId: $rqId}, _set: {fileVersionId: $verId})"`
	}

	variables := map[string]interface{}{
		"rqId":  runConfig[0].RequestId,
		"verId": runConfig[0].FileVersionId,
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
			UpdatedAt string
		} `graphql:"update_import_runtime_by_pk(pk_columns: {requestId: $rqId}, _set: {status: $status})"`
	}

	variables := map[string]interface{}{
		"rqId":   graphql.String(requestId),
		"status": graphql.String(status),
	}

	log.Println(variables)

	if err := a.GraphqlClient.Mutate(context.Background(), &mutation, variables); err != nil {
		log.Fatal(err)
		return err
	}

	log.Println(mutation)

	return nil
}

func (a *Activities) ParseCSV(filekey string) ([]string, error) {
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

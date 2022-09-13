package workflow

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"io"
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

func (a *Activities) GetObject(filekey string) ([][]string, error) {
	object, err := a.MinioClient.GetObject(os.Getenv("MINIO_BUCKET_NAME"), filekey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	var lines [][]string
	var offset int64

	scanner := bufio.NewScanner(bufio.NewReader(object))
	counter := 0

	for scanner.Scan() {
		counter++
	}

	for i := 0; i <= 5; i++ {
		row, _ := bufio.NewReader(object).ReadSlice('\n')
		offset += int64(len(row))
		object.Seek(offset, 0)
	}

	reader := csv.NewReader(object)

	for i := 1; i <= counter-7; i++ {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func (a *Activities) ParseLine(filekey string) ([]string, error) {
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

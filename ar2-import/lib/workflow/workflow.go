package workflow

import (
	"context"
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

func (a *Activities) ReadConfigTable(input string) (UploadTypeConfiguration, error) {
	var q struct {
		UploadTypeConfiguration `graphql:"import_config_config(where: {uploadType: {_eq: $configUploadType}})"`
	}

	variables := map[string]interface{}{
		"configUploadType": graphql.String(input),
	}

	if err := a.GraphQlClient.Query(context.Background(), &q, variables); err != nil {
		log.Fatal(err)
		return nil, err
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

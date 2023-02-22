package utils

import (
	"context"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"logur.dev/logur"
)

type MinioConf struct {
	Endpoint        string
	UseSSL          bool
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicHostName  string
}

func GetMinioClient(config MinioConf, logger logur.KVLoggerFacade) *minio.Client {
	logger.Info("--- Connecting to MinIO ---")
	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.AccessKeyID,
			config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		logger.Error("Unable to connect to filestore: %s\n", err)
		os.Exit(1)
	}
	err = minioClient.MakeBucket(context.Background(),
		config.BucketName,
		minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(),
			config.BucketName)
		if errBucketExists != nil || !exists {
			logger.Error("Unable to create bucket: %s\n", err)
			os.Exit(1)
		}
	}
	logger.Info("Initialized Minio Client:", "minioEndpoint", config.Endpoint)
	return minioClient
}

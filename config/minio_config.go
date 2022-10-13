package config

import (
	"context"
	"fmt"
	"log"
	"minio/model"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var Events []model.Event

type MinioConfig struct {
	MinioClient *minio.Client
}

func NewMinioConfig() *MinioConfig {
	endpoint := "34.118.67.177:32725"
	accessKeyID := "admin"
	secretAccessKey := "xPnmKkFC8u"
	useSSL := false

	var err error
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	go listenNotification(minioClient)
	if err != nil {
		panic(err)
	}
	return &MinioConfig{MinioClient: minioClient}
}

func listenNotification(minioClient *minio.Client) {
	fmt.Println("listening notif...")
	for notificationInfo := range minioClient.ListenNotification(context.Background(), "", "", []string{
		"s3:BucketCreated:*",
		"s3:BucketRemoved:*",
		"s3:ObjectCreated:*",
		// "s3:ObjectAccessed:*",
		"s3:ObjectAccessed:Get",
		"s3:ObjectRemoved:*",
	}) {
		if notificationInfo.Err != nil {
			log.Fatalln(notificationInfo.Err)
		}
		log.Println("************************************")
		event := model.Event{
			EventName:  notificationInfo.Records[0].EventName,
			ObjectName: notificationInfo.Records[0].S3.Object.Key,
			BucketName: notificationInfo.Records[0].S3.Bucket.Name,
			EventTime:  notificationInfo.Records[0].EventTime,
		}
		Events = append(Events, event)
		log.Println("Event: ", event)
	}
}

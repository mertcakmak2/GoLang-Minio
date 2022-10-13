package config

import (
	"context"
	"fmt"
	"log"
	"minio/model"

	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var Events []model.Event

type MinioConfig struct {
	MinioClient      *minio.Client
	MinioAdminClient *madmin.AdminClient
}

func NewMinioConfig() *MinioConfig {
	// endpoint := "34.118.67.177:32725"
	// accessKeyID := "admin"
	// secretAccessKey := "xPnmKkFC8u"
	// useSSL := false

	endpoint := "localhost:9000"
	accessKeyID := "minio"
	secretAccessKey := "minio123"
	useSSL := false

	var err error
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}

	minioAdminClient, err := madmin.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		fmt.Println(err)
	}
	st, err := minioAdminClient.ServerInfo(context.Background())
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Server info: ", st)

	go listenNotification(minioClient)
	return &MinioConfig{MinioClient: minioClient, MinioAdminClient: minioAdminClient}
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
		notification := notificationInfo.Records[0]
		event := model.Event{
			EventName:  notification.EventName,
			ObjectName: notification.S3.Object.Key,
			BucketName: notification.S3.Bucket.Name,
			EventTime:  notification.EventTime,
		}
		Events = append(Events, event)
		log.Println("Event: ", event)
	}
}

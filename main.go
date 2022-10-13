package main

import (
	"context"
	"fmt"
	"log"
	"minio/config"
	"minio/model"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

var minioClient *minio.Client
var ctx = context.Background()

func main() {
	router := gin.Default()

	// minioConnection()
	minioConfig := config.NewMinioConfig()
	minioClient = minioConfig.MinioClient
	// object endpoints
	router.GET("/api/objects/:bucket/:name", getObject)               // http://localhost:8080/api/objects/bucketname/Screenshot_1.png
	router.GET("/api/objects/download/:bucket/:name", downloadObject) // http://localhost:8080/api/objects/download/bucketname/Screenshot_26.png
	router.DELETE("/api/objects/:bucket/:name", deleteObject)         // http://localhost:8080/api/objects/bucketname/Screenshot_1.png
	router.POST("/api/objects/:bucket", uploadFile)                   // http://localhost:8080/api/objects/bucketname with (form data file key)
	// bucket endpoints
	router.POST("/api/buckets/:bucket", makeBucket)     //	http://localhost:8080/api/buckets/new-bucket
	router.DELETE("/api/buckets/:bucket", deleteBucket) //	http://localhost:8080/api/buckets/new-bucket
	// notificaton endpoint
	router.GET("/api/notifications", getNotifications) //	http://localhost:8080/api/notifications
	router.Run(":8080")
}

func getObject(c *gin.Context) {
	name := c.Param("name")
	bucket := c.Param("bucket")
	object, err := minioClient.GetObject(context.Background(), bucket, name, minio.GetObjectOptions{})
	if err != nil {
		panic(err)
	}
	data, _ := object.Stat()
	size := data.Size
	if data.Size == 0 {
		c.JSON(http.StatusNoContent, gin.H{"error": "File not found"})
		return
	}
	extraHeaders := map[string]string{
		"Content-Disposition": "inline; filename=" + name,
	}
	c.DataFromReader(http.StatusOK, size, "MediaType.IMAGE_PNG", object, extraHeaders)
}

func downloadObject(c *gin.Context) {
	name := c.Param("name")
	bucket := c.Param("bucket")
	object, err := minioClient.GetObject(context.Background(), bucket, name, minio.GetObjectOptions{})
	if err != nil {
		panic(err)
	}
	data, _ := object.Stat()
	if data.Size == 0 {
		c.JSON(http.StatusNoContent, gin.H{"error": "File not found"})
		return
	}
	extraHeaders := map[string]string{
		"Content-Disposition": "inline; filename=" + name,
	}
	c.DataFromReader(http.StatusOK, data.Size, "application/octet-stream", object, extraHeaders)
}

func uploadFile(c *gin.Context) {
	log.Println("uploading...")
	bucket := c.Param("bucket")
	_, header, err := c.Request.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("file err : %s", err.Error()))
		return
	}
	filename := header.Filename
	contentType := header.Header.Get("Content-Type")
	size := header.Size
	buffer, err := header.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"filepath": "buffer error"})
		return
	}
	log.Println("puting...")
	info, err := minioClient.PutObject(ctx, bucket, filename, buffer, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"filepath": "put object error"})
		return
	}
	log.Println("uploaded...")
	c.JSON(http.StatusCreated, gin.H{"filepath": info.Key, "header": header.Header.Get("Content-Type")})
}

func deleteObject(c *gin.Context) {
	name := c.Param("name")
	bucket := c.Param("bucket")

	if err := minioClient.RemoveObject(ctx, bucket, name, minio.RemoveObjectOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed while deleting."})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{"message": "Object deleted."})
}

func makeBucket(c *gin.Context) {
	bucket := c.Param("bucket")
	location := "us-east-1"
	err := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, bucket)
		if errBucketExists == nil && exists {
			c.JSON(http.StatusConflict, gin.H{"error": "we already own " + bucket})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	} else {
		c.JSON(http.StatusCreated, gin.H{"message": "Successfully created " + bucket})
		return
	}
}

func deleteBucket(c *gin.Context) {
	bucket := c.Param("bucket")
	if err := minioClient.RemoveBucket(ctx, bucket); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed while deleting"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Bucket deleted."})
}

func getNotifications(c *gin.Context) {
	events := config.Events
	if len(events) > 0 {
		c.JSON(http.StatusOK, events)
		return
	}
	c.JSON(http.StatusOK, []model.Event{})
}

// func minioConnection() {
// 	// Kubernetes Minio
// 	endpoint := "34.118.67.177:32725"
// 	accessKeyID := "admin"
// 	secretAccessKey := "xPnmKkFC8u"
// 	useSSL := false

// 	var err error
// 	minioClient, err = minio.New(endpoint, &minio.Options{
// 		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
// 		Secure: useSSL,
// 	})
// 	go listenNotification(minioClient)
// 	if err != nil {
// 		panic(err)
// 	}
// }

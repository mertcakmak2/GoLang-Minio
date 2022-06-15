package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {

	router := gin.Default()

	router.POST("/upload-file", uploadFile)
	router.POST("/make-bucket", makeBucket)
	router.DELETE("/delete-object", deleteObject)
	router.Run(":8080")
}

func uploadFile(c *gin.Context) {
	minioClient, err := minioConnection()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "connection error"})
		return
	}

	bucket, isThere := c.GetQuery("bucket")
	if !isThere {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bucket query doesn't exist "})
		return
	}

	ctx := context.Background()
	bucketName := bucket

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

	info, err := minioClient.PutObject(ctx, bucketName, filename, buffer, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"filepath": "put object error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"filepath": info.Key, "header": header.Header.Get("Content-Type")})
}

func deleteObject(c *gin.Context) {

	minioClient, err := minioConnection()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "connection error"})
		return
	}

	bucket, isThere := c.GetQuery("bucket")
	if !isThere {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bucket query doesn'e exist "})
		return
	}

	object, isThere := c.GetQuery("object")
	if !isThere {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bucket query doesn'e exist "})
		return
	}

	ctx := context.Background()
	bucketName := bucket
	minioClient.RemoveObject(ctx, bucketName, object, minio.RemoveObjectOptions{})

}

func makeBucket(c *gin.Context) {

	minioClient, err := minioConnection()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "connection error"})
		return
	}

	bucket, isThere := c.GetQuery("bucket")
	if !isThere {
		c.JSON(http.StatusOK, gin.H{"error": "bucket query doesnt'e exist "})
		return
	}

	ctx := context.Background()
	bucketName := bucket
	location := "us-east-1"

	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			c.JSON(http.StatusConflict, gin.H{"error": "we already own " + bucketName})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	} else {
		c.JSON(http.StatusCreated, gin.H{"error": "Successfully created %s\n"})
		return
	}
}

func minioConnection() (*minio.Client, error) {

	endpoint := "localhost:9000"
	accessKeyID := "minio"
	secretAccessKey := "minio123"
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	} else {
		return minioClient, nil
	}

}

package main

import (
	"context"
	"fmt"
	"minio/config"
	"minio/model"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
)

var minioClient *minio.Client
var minioAdminClient *madmin.AdminClient
var ctx = context.Background()

func main() {
	router := gin.Default()

	minioConfig := config.NewMinioConfig()
	minioClient = minioConfig.MinioClient
	minioAdminClient = minioConfig.MinioAdminClient

	// object endpoints
	router.GET("/api/objects/:bucket/:name", getObject)               // http://localhost:8080/api/objects/bucketname/Screenshot_1.png
	router.GET("/api/objects/download/:bucket/:name", downloadObject) // http://localhost:8080/api/objects/download/bucketname/Screenshot_26.png
	router.DELETE("/api/objects/:bucket/:name", deleteObject)         // http://localhost:8080/api/objects/bucketname/Screenshot_1.png
	router.POST("/api/objects/:bucket", uploadFile)                   // http://localhost:8080/api/objects/bucketname with (form data file key)

	// bucket endpoints
	router.POST("/api/buckets/:bucket", makeBucket)     // http://localhost:8080/api/buckets/new-bucket
	router.DELETE("/api/buckets/:bucket", deleteBucket) // http://localhost:8080/api/buckets/new-bucket
	router.GET("/api/buckets", retrieveBuckets)         // http://localhost:8080/api/buckets

	// groups endpoints
	router.GET("/api/groups", retrieveGroups)       // http://localhost:8080/api/groups
	router.POST("/api/groups/:name", addGroup)      // http://localhost:8080/api/groups/group-test
	router.DELETE("/api/groups/:name", removeGroup) // http://localhost:8080/api/groups/group-test

	// users endpoints
	router.POST("/api/users", createUser)             // http://localhost:8080/api/users
	router.DELETE("/api/users/:username", deleteUser) // http://localhost:8080/api/users

	// notificaton endpoint
	router.GET("/api/notifications", getNotifications) // http://localhost:8080/api/notifications
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
	info, err := minioClient.PutObject(ctx, bucket, filename, buffer, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"filepath": "put object error"})
		return
	}
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

func retrieveBuckets(c *gin.Context) {
	buckets, err := minioClient.ListBuckets(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, buckets)
}

func retrieveGroups(c *gin.Context) {
	groups, err := minioAdminClient.ListGroups(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func addGroup(c *gin.Context) {
	group := c.Param("name")
	groupAddRemove := madmin.GroupAddRemove{Group: group, IsRemove: false, Status: madmin.GroupEnabled}
	err := minioAdminClient.UpdateGroupMembers(ctx, groupAddRemove)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, "Group created.")
}

func removeGroup(c *gin.Context) {
	group := c.Param("name")
	groupAddRemove := madmin.GroupAddRemove{Group: group, IsRemove: true}
	err := minioAdminClient.UpdateGroupMembers(ctx, groupAddRemove)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, "Group deleted.")
}

func createUser(c *gin.Context) {
	var createUser model.CreateUser
	if err := c.ShouldBindJSON(&createUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := minioAdminClient.AddUser(ctx, createUser.Username, createUser.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, "User created.")
}

func deleteUser(c *gin.Context) {
	username := c.Param("username")
	err := minioAdminClient.RemoveUser(ctx, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, "User deleted.")
}

func getNotifications(c *gin.Context) {
	events := config.Events
	if len(events) > 0 {
		c.JSON(http.StatusOK, events)
		return
	}
	c.JSON(http.StatusOK, []model.Event{})
}

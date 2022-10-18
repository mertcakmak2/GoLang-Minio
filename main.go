package main

import (
	"context"
	"fmt"
	"minio/config"
	"minio/model"
	"net/http"
	"strings"

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

	// objects endpoints
	router.GET("/api/objects/:bucket/:name", getObject)               // http://localhost:8080/api/objects/bucketname/Screenshot_1.png
	router.GET("/api/objects/download/:bucket/:name", downloadObject) // http://localhost:8080/api/objects/download/bucketname/Screenshot_26.png
	router.DELETE("/api/objects/:bucket/:name", deleteObject)         // http://localhost:8080/api/objects/bucketname/Screenshot_1.png
	router.POST("/api/objects/:bucket", uploadFile)                   // http://localhost:8080/api/objects/bucketname with (form data file key)

	// buckets endpoints
	router.POST("/api/buckets/:bucket", makeBucket)     // http://localhost:8080/api/buckets/new-bucket
	router.DELETE("/api/buckets/:bucket", deleteBucket) // http://localhost:8080/api/buckets/new-bucket
	router.GET("/api/buckets", retrieveBuckets)         // http://localhost:8080/api/buckets

	// groups endpoints
	router.GET("/api/groups", retrieveGroups)       // http://localhost:8080/api/groups
	router.POST("/api/groups/:name", addGroup)      // http://localhost:8080/api/groups/group-test
	router.DELETE("/api/groups/:name", removeGroup) // http://localhost:8080/api/groups/group-test

	// users endpoints
	router.POST("/api/users", createUser)             // http://localhost:8080/api/users
	router.GET("/api/users", retrieveUsers)           // http://localhost:8080/api/users
	router.GET("/api/users/:name", getUserInfo)       // http://localhost:8080/api/users/user1
	router.DELETE("/api/users/:username", deleteUser) // http://localhost:8080/api/users/user1

	// notificatons endpoints
	router.GET("/api/notifications", getNotifications) // http://localhost:8080/api/notifications

	// policies endpoints
	router.GET("/api/policies/:policy/bucket/:bucket", createPolicy) // http://localhost:8080/api/policies/a-policy/bucket/a-bucket
	// router.GET("/api/policies/:policy/user/:user", setPolicyToUser)    // http://localhost:8080/api/policies/a-policy/user/a-user
	router.GET("/api/policies/:policy/group/:group", setPolicyToGroup) // http://localhost:8080/api/policies/a-policy/group/a-group
	router.GET("/api/policies", retrievePolicies)                      // http://localhost:8080/api/policies/a-policy/bucket/a-bucket

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
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Successfully created " + bucket})
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

func getUserInfo(c *gin.Context) {
	username := c.Param("name")
	userInfo, _ := minioAdminClient.GetUserInfo(context.Background(), username)
	user := map[string]interface{}{
		"username": username,
		"status":   userInfo.Status,
		"groups":   userInfo.MemberOf,
		"roles":    userInfo.PolicyName,
	}
	c.JSON(http.StatusOK, user)
}

func retrieveUsers(c *gin.Context) {
	users, err := minioAdminClient.ListUsers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func createPolicy(c *gin.Context) {
	bucket := c.Param("bucket")
	policyName := c.Param("policy")
	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:*"],"Resource":["arn:aws:s3:::BUCKET-NAME/*"]}]}`
	newPolicy := strings.Replace(policy, "BUCKET-NAME", bucket, -1)
	err := minioAdminClient.AddCannedPolicy(ctx, policyName+"-go", []byte(newPolicy)) // yeni policy oluşturma
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": newPolicy})
}

// func setPolicyToUser(c *gin.Context) {
// 	user := c.Param("user")
// 	policyName := c.Param("policy")
// 	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:*"],"Resource":["arn:aws:s3:::BUCKET-NAME/*"]}]}`
// 	newPolicy := strings.Replace(policy, "BUCKET-NAME", bucket, -1)
// 	// fmt.Println([]byte(newPolicy))
// 	err := minioAdminClient.SetPolicy(ctx, policy, user, false) // user'a policy atama
// 	// err := minioAdminClient.AddCannedPolicy(ctx, bucket+"-policy-go", []byte(newPolicy)) // yeni policy oluşturma
// 	// err = minioClient.SetBucketPolicy(ctx, bucket, newPolicy)
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{"msg": "set policy to user"})
// }

func setPolicyToGroup(c *gin.Context) {
	group := c.Param("group")
	policy := c.Param("policy")
	desc, err1 := minioAdminClient.GetGroupDescription(ctx, "a-group")
	if err1 != nil {
		fmt.Println(err1.Error())
	}
	fmt.Println(desc)
	err := minioAdminClient.SetPolicy(ctx, policy, group, true) // group'a policy atama
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "set policy to group"})
}

func retrievePolicies(c *gin.Context) {
	val, err := minioAdminClient.ListCannedPolicies(ctx)
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": val})
}

func getNotifications(c *gin.Context) {
	events := config.Events
	if len(events) > 0 {
		c.JSON(http.StatusOK, events)
		return
	}
	c.JSON(http.StatusOK, []model.Event{})
}

package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"gopkg.in/h2non/gock.v1"
	"bytes"
	"errors"
	"github.com/spf13/afero"
)

func TestBucketService_DeleteFile(t *testing.T) {


	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Delete("/storage/v1/b/foobar/o/foobar_file").
		Reply(200)

	err := getService(t).DeleteFile("foobar_file")

	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}

func TestBucketService_RemoveBucketACL(t *testing.T) {


	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Delete("/storage/v1/b/foobar/acl/user-service_account_test").
		Reply(200)

	err := getService(t).RemoveBucketACL("service_account_test")

	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}

func TestBucketService_SetBucketACL(t *testing.T) {
	response := `
	{
  "kind": "storage#bucketAccessControl",
  "id": "aclID",
  "selfLink": "selfLink",
  "bucket": "foobar",
  "entity": "asa",
  "role": "dummy",
  "email": "foobar@foo",
  "entityId": "id",
  "domain": "a",
  "projectTeam": {
    "projectNumber": "s",
    "team": "s"
  },
  "etag": "s"
}`

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Post("/storage/v1/b/foobar/acl").
		MatchType("json").
		JSON(`{"email":"service_account_test","entity":"user-service_account_test","role":"READER"}`).
		Reply(200).
		JSON(response)

	err := getService(t).SetBucketACL("service_account_test", "READER")

	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}

func TestBucketService_DownLoadFile(t *testing.T) {
	response := `
	{
  "kind": "storage#bucketAccessControl",
  "id": "aclID",
  "selfLink": "selfLink",
  "bucket": "foobar",
  "entity": "asa",
  "role": "dummy",
  "email": "foobar@foo",
  "entityId": "id",
  "domain": "a",
  "projectTeam": {
    "projectNumber": "s",
    "team": "s"
  },
  "etag": "s"
}`

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Post("/storage/v1/b/foobar/acl").
		MatchType("json").
		JSON(`{"email":"service_account_test","entity":"user-service_account_test","role":"READER"}`).
		Reply(200).
		JSON(response)

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Get("/storage/v1/b/foobar/o/dummy.foobar").
		Reply(200).
		JSON(`{"mediaLink": "https://never.work.local/download"}`)

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://never.work.local").
		Get("/download").
		Reply(200).
		BodyString("file_content")


	result, err := getService(t).DownLoadFile("dummy.foobar", "service_account_test")

	buf := new(bytes.Buffer)
	buf.ReadFrom(result)

	assert.NoError(t, err)
	assert.Equal(t, "file_content", buf.String())
	assert.True(t, gock.IsDone())
}

func TestBucketService_DownLoadFileWithErrorForSetAcl(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Post("/storage/v1/b/foobar/acl").
		MatchType("json").
		JSON(`{"email":"service_account_test","entity":"user-service_account_test","role":"READER"}`).
		ReplyError(errors.New("SetBucketACLError"))

	_, err := getService(t).DownLoadFile("dummy.foobar", "service_account_test")


	assert.EqualError(t, err, "Post https://www.googleapis.com/storage/v1/b/foobar/acl?alt=json: SetBucketACLError")
	assert.True(t, gock.IsDone())
}

func TestBucketService_DownLoadFileWithErrorToGetObjectInformation(t *testing.T) {
	response := `
	{
  "kind": "storage#bucketAccessControl",
  "id": "aclID",
  "selfLink": "selfLink",
  "bucket": "foobar",
  "entity": "asa",
  "role": "dummy",
  "email": "foobar@foo",
  "entityId": "id",
  "domain": "a",
  "projectTeam": {
    "projectNumber": "s",
    "team": "s"
  },
  "etag": "s"
}`

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Post("/storage/v1/b/foobar/acl").
		MatchType("json").
		JSON(`{"email":"service_account_test","entity":"user-service_account_test","role":"READER"}`).
		Reply(200).
		JSON(response)

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Get("/storage/v1/b/foobar/o/dummy.foobar").
		ReplyError(errors.New("ObjectInfoError"))

	_, err := getService(t).DownLoadFile("dummy.foobar", "service_account_test")


	assert.EqualError(t, err, "Get https://www.googleapis.com/storage/v1/b/foobar/o/dummy.foobar?alt=json: ObjectInfoError")
	assert.True(t, gock.IsDone())
}

func TestBucketService_DownLoadFileWithDownloadError(t *testing.T) {
	response := `
	{
  "kind": "storage#bucketAccessControl",
  "id": "aclID",
  "selfLink": "selfLink",
  "bucket": "foobar",
  "entity": "asa",
  "role": "dummy",
  "email": "foobar@foo",
  "entityId": "id",
  "domain": "a",
  "projectTeam": {
    "projectNumber": "s",
    "team": "s"
  },
  "etag": "s"
}`

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Post("/storage/v1/b/foobar/acl").
		MatchType("json").
		JSON(`{"email":"service_account_test","entity":"user-service_account_test","role":"READER"}`).
		Reply(200).
		JSON(response)

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Get("/storage/v1/b/foobar/o/dummy.foobar").
		Reply(200).
		JSON(`{"mediaLink": "https://never.work.local/download"}`)

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://never.work.local").
		Get("/download").
		ReplyError(errors.New("DownloadError"))


	_, err := getService(t).DownLoadFile("dummy.foobar", "service_account_test")


	assert.EqualError(t, err, "Get https://never.work.local/download: DownloadError")
	assert.True(t, gock.IsDone())
}

func TestBucketService_UploadFile(t *testing.T) {

	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "original.file", []byte("content_upload"), 0644)

	oldFileSystem := fileSystem
	fileSystem = appFS

	defer func() { fileSystem = oldFileSystem }()
	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.BodyTypes = append(gock.BodyTypes, "multipart/related")

	gock.New("https://www.googleapis.com").
		Post("/upload/storage/v1/b/foobar/o").
		MatchType("multipart/related").
		Reply(200).
		JSON(`{}`)

	err := getService(t).UploadFile("target.file", "original.file", "service_account_test")

	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}

func TestBucketService_UploadFileWithNotExistingFile(t *testing.T) {

	appFS := afero.NewMemMapFs()
	// create test files and directories
	afero.WriteFile(appFS, "original.file", []byte("content_upload"), 0644)

	oldFileSystem := fileSystem
	fileSystem = appFS

	defer func() { fileSystem = oldFileSystem }()
	defer gock.Off() // Flush pending mocks after test execution

	err := getService(t).UploadFile("target.file", "original_2.file", "service_account_test")

	assert.EqualError(t, err, "open original_2.file: file does not exist")
	assert.True(t, gock.IsDone())
}

func getService(t *testing.T) BucketServiceInterface {
	bucketService, err := new(Builder).GetStorageService("foobar")
	assert.NoError(t, err)

	return bucketService
}

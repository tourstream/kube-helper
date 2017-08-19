package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"gopkg.in/h2non/gock.v1"
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

func getService(t *testing.T) BucketServiceInterface {
	bucketService, err := new(Builder).GetStorageService("foobar")
	assert.NoError(t, err)

	return bucketService
}

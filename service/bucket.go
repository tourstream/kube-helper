package service

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"

	StorageClient "cloud.google.com/go/storage"
	"google.golang.org/api/storage/v1"
	"context"
)

type BucketServiceInterface interface {
	DownLoadFile(filename string, serviceAccountEmailAddress string) (*os.File, error)
	SetBucketACL(serviceAccountEmailAddress string, role string) error
	RemoveBucketACL(serviceAccount string) error
	DeleteFile(filename string) error
	UploadFile(targetFilename string, originalFilename string, serviceAccountEmailAddress string) error
}

type bucketService struct {
	storageService *storage.Service
	httpClient     *http.Client
	bucket         string
	storageClient  *StorageClient.Client
}

func NewBucketService(bucket string, client *http.Client, storageService *storage.Service, storageClient *StorageClient.Client) BucketServiceInterface {
	b := new(bucketService)
	b.httpClient = client
	b.bucket = bucket
	b.storageService = storageService
	b.storageClient = storageClient

	return b
}

func (b *bucketService) DownLoadFile(filename string, serviceAccountEmailAddress string) (*os.File, error) {

	err := b.SetBucketACL(serviceAccountEmailAddress, "READER")

	if err != nil {
		return nil, err
	}

	bucket, err := b.storageService.Objects.Get(b.bucket, filename).Do()

	if err != nil {
		return nil, err
	}

	tmpfile, err := ioutil.TempFile("", filename)
	if err != nil {
		return nil, err
	}

	defer os.Remove(tmpfile.Name()) //cleanuo

	response, err := b.httpClient.Get(bucket.MediaLink)

	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	_, err = io.Copy(tmpfile, response.Body)

	if err != nil {
		return nil, err
	}

	return tmpfile, err
}

func (b *bucketService) DeleteFile(filename string) error {
	return b.storageService.Objects.Delete(b.bucket, filename).Do()
}

func (b *bucketService) SetBucketACL(serviceAccountEmailAddress string, role string) error {
	_, err := b.storageService.BucketAccessControls.Insert(b.bucket, &storage.BucketAccessControl{
		Email:  serviceAccountEmailAddress,
		Entity: "user-" + serviceAccountEmailAddress,
		Role:   role,
	}).Do()

	if err != nil {
		return err
	}

	return nil
}

func (b *bucketService) RemoveBucketACL(serviceAccountEmailAddress string) error {
	return b.storageService.BucketAccessControls.Delete(b.bucket, "user-"+serviceAccountEmailAddress).Do()
}

func (b *bucketService) UploadFile(targetFilename string, originalFilename string, serviceAccountEmailAddress string) error {

	w := b.storageClient.Bucket(b.bucket).Object(targetFilename).NewWriter(context.Background())
	w.ACL = []StorageClient.ACLRule{{Entity: StorageClient.ACLEntity("user-" + serviceAccountEmailAddress), Role: StorageClient.RoleReader}}

	file, err := os.Open(originalFilename)

	if err != nil {
		return err
	}

	_, err = io.Copy(w, file)

	if err != nil {
		return err
	}

	return w.Close()
}

package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"kube-helper/loader"
	"gopkg.in/h2non/gock.v1"
	"errors"
	"kube-helper/model"
)

func TestImages_HasTag(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Get("/v2/project/container/manifests/branch-tag").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	result, err := imageService.HasTag(loader.Cleanup{ImagePath: "google-registry/project/container"}, "branch-tag")

	assert.NoError(t, err)
	assert.True(t, result)

}

func TestImages_HasTagWithWrongStatusCode(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Get("/v2/project/container/manifests/branch-tag").
		Reply(404).
		JSON(map[string]string{"foo": "bar"})

	result, err := imageService.HasTag(loader.Cleanup{ImagePath: "google-registry/project/container"}, "branch-tag")

	assert.NoError(t, err)
	assert.False(t, result)

}

func TestImages_List(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution

	response := `
	{
	"child": [],
	"manifest": {
		"sha256:1": {
			"imageSizeBytes": "3607047",
			"layerId": "layer-1",
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"tag": [],
			"timeCreatedMs": "1489090775466",
			"timeUploadedMs": "1489090878674"
		},
		"sha256:2": {
			"imageSizeBytes": "3607201",
			"layerId": "layer-2",
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"tag": ["1.08"],
			"timeCreatedMs": "1489698289474",
			"timeUploadedMs": "1489702107448"
		},
		"sha256:3": {
			"imageSizeBytes": "3675898",
			"layerId": "layer-3",
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"tag": [],
			"timeCreatedMs": "1484065818532",
			"timeUploadedMs": "1484065939440"
		},
		"sha256:4": {
			"imageSizeBytes": "5994541",
			"layerId": "layer-4",
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"tag": ["1.09", "latest"],
			"timeCreatedMs": "1490900872959",
			"timeUploadedMs": "1496340933854"
		},

		"sha256:5": {
			"imageSizeBytes": "4754998",
			"layerId": "layer-5",
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"tag": ["1.10"],
			"timeCreatedMs": "1498512067506",
			"timeUploadedMs": "1498514749494"
		},
		"sha256:6": {
			"imageSizeBytes": "2413760",
			"layerId": "layer-6",
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"tag": ["test-249c4560aac8"],
			"timeCreatedMs": "1475860677921",
			"timeUploadedMs": "1484252668648"
		}
	},
	"name": "cloudsql-docker/gce-proxy",
	"tags": ["1.08", "1.09", "1.10", "latest", "test-249c4560aac8"]
}
	`

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Get("/v2/project/container/tags/list").
		Reply(200).
		JSON(response)

	result, err := imageService.List(loader.Cleanup{ImagePath: "google-registry/project/container"})

	expected := new(model.TagCollection)
	expected.Name = "cloudsql-docker/gce-proxy"
	expected.Manifests = map[string]model.Manifest{
		"sha256:1": {
			LayerId:       "layer-1",
			TimeCreatedMs: int64(1489090775466),
			Tags:          []string{},
		},
		"sha256:2": {
			LayerId:       "layer-2",
			TimeCreatedMs: int64(1489698289474),
			Tags:          append(make([]string, 0, 4), "1.08"),
		},
		"sha256:3": {
			LayerId:       "layer-3",
			TimeCreatedMs: int64(1484065818532),
			Tags:          []string{},
		},
		"sha256:4": {
			LayerId:       "layer-4",
			TimeCreatedMs: int64(1490900872959),
			Tags:          append(make([]string, 0, 4), "1.09", "latest"),
		},
		"sha256:5": {
			LayerId:       "layer-5",
			TimeCreatedMs: int64(1498512067506),
			Tags:          append(make([]string, 0, 4), "1.10"),
		},
		"sha256:6": {
			LayerId:       "layer-6",
			TimeCreatedMs: int64(1475860677921),
			Tags:          append(make([]string, 0, 4), "test-249c4560aac8"),
		},

	}
	expected.SortedManifests = append(make([]model.ManifestPair, 0, 8),
		model.ManifestPair{
			Key: "sha256:5",
			Value: model.Manifest{
				LayerId:       "layer-5",
				TimeCreatedMs: int64(1498512067506),
				Tags:          append(make([]string, 0, 4), "1.10"),
			},
		},
		model.ManifestPair{
			Key: "sha256:4",
			Value: model.Manifest{
				LayerId:       "layer-4",
				TimeCreatedMs: int64(1490900872959),
				Tags:          append(make([]string, 0, 4), "1.09", "latest"),
			},
		},
		model.ManifestPair{
			Key: "sha256:2",
			Value: model.Manifest{
				LayerId:       "layer-2",
				TimeCreatedMs: int64(1489698289474),
				Tags:          append(make([]string, 0, 4), "1.08"),
			},
		},
		model.ManifestPair{
			Key: "sha256:1",
			Value: model.Manifest{
				LayerId:       "layer-1",
				TimeCreatedMs: int64(1489090775466),
				Tags:          []string{},
			},
		},
		model.ManifestPair{
			Key: "sha256:3",
			Value: model.Manifest{
				LayerId:       "layer-3",
				TimeCreatedMs: int64(1484065818532),
				Tags:          []string{},
			},
		},
		model.ManifestPair{
			Key: "sha256:6",
			Value: model.Manifest{
				LayerId:       "layer-6",
				TimeCreatedMs: int64(1475860677921),
				Tags:          append(make([]string, 0, 4), "test-249c4560aac8"),
			},
		})

	assert.NoError(t, err)
	assert.Equal(t, expected, result)

}

func TestImages_ListWithNoValidJson(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution

	response := `No Valid JSON
	`

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Get("/v2/project/container/tags/list").
		Reply(200).
		BodyString(response)

	result, err := imageService.List(loader.Cleanup{ImagePath: "google-registry/project/container"})

	assert.EqualError(t, err, "invalid character 'N' looking for beginning of value")
	assert.Nil(t, result)
}

func TestImages_ListWithErrorToGetList(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Get("/v2/project/container/tags/list").
		ReplyError(errors.New("ListError"))

	result, err := imageService.List(loader.Cleanup{ImagePath: "google-registry/project/container"})

	assert.EqualError(t, err, "Get https://google-registry/v2/project/container/tags/list: ListError")
	assert.Nil(t, result)

}


func TestImages_ListWithWrongStatusCode(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution



	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Get("/v2/project/container/tags/list").
		Reply(400)

	result, err := imageService.List(loader.Cleanup{ImagePath: "google-registry/project/container"})

	expected := new(model.TagCollection)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)

}

func TestImages_DeleteManifest(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Delete("/v2/project/container/manifests/branch-tag").
		Reply(200)

	err := imageService.DeleteManifest(loader.Cleanup{ImagePath: "google-registry/project/container"}, "branch-tag")

	assert.NoError(t, err)
}

func TestImages_DeleteManifestWithError(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Delete("/v2/project/container/manifests/branch-tag").
		ReplyError(errors.New("DeleteError"))

	err := imageService.DeleteManifest(loader.Cleanup{ImagePath: "google-registry/project/container"}, "branch-tag")

	assert.EqualError(t, err, "Delete https://google-registry/v2/project/container/manifests/branch-tag: DeleteError")
}

func TestImages_Untag(t *testing.T) {

	imageService := new(Images)

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://google-registry").
		Delete("/v2/project/container/manifests/branch-tag").
		Reply(200)

	err := imageService.Untag(loader.Cleanup{ImagePath: "google-registry/project/container"}, "branch-tag")

	assert.NoError(t, err)
}

package image

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"kube-helper/loader"
	"kube-helper/model"
	"sort"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
)

type ImagesInterface interface {
	List(config loader.Cleanup) (*model.TagCollection, error)
	HasTag(config loader.Cleanup, tag string) (bool, error)
	Untag(config loader.Cleanup, tag string) error
	DeleteManifest(config loader.Cleanup, manifest string) error
}

type images struct {
	client *http.Client
}

const manifestPath = "https://%s/v2/%s/%s/manifests/%s"

func NewImagesService() (*images, error) {
	ctx := context.Background()
	client, err := google.DefaultClient(ctx)

	if err != nil {
		return nil, err
	}

	k := images{client}
	return &k, nil
}

func (i *images) HasTag(config loader.Cleanup, tag string) (bool, error) {
	imagePath := strings.Split(config.ImagePath, "/")

	resp, err := i.client.Get(fmt.Sprintf(manifestPath, imagePath[0], imagePath[1], imagePath[2], tag))

	if err != nil {
		return false, err
	}

	if resp.StatusCode == 200 { // OK
		return true, nil
	}

	return false, nil

}

func (i *images) List(config loader.Cleanup) (*model.TagCollection, error) {
	imagePath := strings.Split(config.ImagePath, "/")

	resp, err := i.client.Get(fmt.Sprintf("https://%s/v2/%s/%s/tags/list", imagePath[0], imagePath[1], imagePath[2]))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var s = new(model.TagCollection)

	if resp.StatusCode == 200 { // OK
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(bodyBytes, &s)
		if err != nil {
			return nil, err
		}
	}

	var ss []model.ManifestPair
	for k, v := range s.Manifests {
		ss = append(ss, model.ManifestPair{Key: k, Value: v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value.TimeCreatedMs > ss[j].Value.TimeCreatedMs
	})

	s.SortedManifests = ss

	return s, nil
}

func (i *images) Untag(config loader.Cleanup, tag string) error {
	return i.DeleteManifest(config, tag)
}

func (i *images) DeleteManifest(config loader.Cleanup, manifest string) error {
	imagePath := strings.Split(config.ImagePath, "/")

	req, err := http.NewRequest("DELETE", fmt.Sprintf(manifestPath, imagePath[0], imagePath[1], imagePath[2], manifest), nil)
	if err != nil {
		return err
	}
	_, err = i.client.Do(req)

	return err
}

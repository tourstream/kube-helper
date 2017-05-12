package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/oauth2/google"
	"kube-helper/loader"
)

type Manifest struct {
	LayerId string   `json:"layerId"`
	Tags    []string `json:"tag"`
}

type TagCollection struct {
	Name      string
	Manifests map[string]Manifest `json:"manifest"`
}

type ImagesInterface interface {
	List(config loader.Cleanup) (*TagCollection, error)
	Untag(tag string) error
	DeleteManifest(manifest string) error
}

type Images struct {
	client *http.Client
}

func (i *Images) List(config loader.Cleanup) (*TagCollection, error) {
	err := i.setClient()

	if err != nil {
		return nil, err
	}

	imagePath := strings.Split(config.ImagePath, "/")

	resp, err := i.client.Get(fmt.Sprintf("https://%s/v2/%s/%s/tags/list", imagePath[0], imagePath[1], imagePath[2]))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var s = new(TagCollection)

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

	return s, nil
}

func (i *Images) Untag(tag string) error {
	return i.DeleteManifest(tag)
}

func (i *Images) DeleteManifest(manifest string) error {
	err := i.setClient()

	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://eu.gcr.io/v2/n2170-container-engine-spike/php-app/manifests/%s", manifest), nil)
	if err != nil {
		return err
	}
	_, err = i.client.Do(req)

	return err
}

func (i *Images) setClient() error {
	ctx := context.Background()

	client, err := google.DefaultClient(ctx)

	if err != nil {
		return err
	}

	i.client = client

	return nil
}

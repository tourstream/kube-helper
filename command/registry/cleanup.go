package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"kube-helper/util"

	"kube-helper/loader"

	"github.com/urfave/cli"
	"golang.org/x/oauth2/google"
)

var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var branchLoader loader.BranchLoaderInterface = new(loader.BranchLoader)

func CmdCleanup(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	manifests, err := getImageTags()

	if err != nil {
		return err
	}
	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return err
	}

	imagesToDelete := []string{}

	for manifestId, manifest := range manifests.Manifests {
		cleanup := true
		for _, tag := range manifest.Tags {
			if strings.HasPrefix(tag, "staging-") == false {
				continue
			}

			if tag == "staging-latest" {
				cleanup = false
				break
			}

			if strings.HasSuffix(tag, "latest") {

				branchName := strings.TrimSuffix(strings.TrimPrefix(tag, "staging-"), "-latest")

				//do not cleanup if branch exists
				if inArray(branches, branchName) {
					cleanup = false
					break
				}

			}
		}

		if cleanup {
			imagesToDelete = append(imagesToDelete, configContainer.Cleanup.ImagePath+"@"+manifestId)

		}
	}

	for _, image := range imagesToDelete {
		otherCmd := exec.Command("gcloud", "beta", "container", "images", "delete", image, "--resolve-tag-to-digest", "--force-delete-tags")
		stdoutStderr, err := otherCmd.CombinedOutput()
		fmt.Printf("Output:%s\n", stdoutStderr)
		util.CheckError(err)
	}

	return nil
}

type Manifest struct {
	LayerId string   `json:"layerId"`
	Tags    []string `json:"tag"`
}

type TagCollection struct {
	Name      string
	Manifests map[string]Manifest `json:"manifest"`
}

func getImageTags() (*TagCollection, error) {
	ctx := context.Background()

	client, err := google.DefaultClient(ctx)

	if err != nil {
		return nil, err
	}

	resp, err := client.Get("https://eu.gcr.io/v2/n2170-container-engine-spike/php-app/tags/list")

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

func inArray(haystack []string, needle string) bool {
	for _, el := range haystack {
		if el == needle {
			return true
		}
	}

	return false
}

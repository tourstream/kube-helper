package registry

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli"
	"kube-helper/loader"
	"kube-helper/service"
)

var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var branchLoader loader.BranchLoaderInterface = new(loader.BranchLoader)
var imagesService service.ImagesInterface = new(service.Images)
var writer io.Writer = os.Stdout

func CmdCleanup(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	manifests, err := imagesService.List(configContainer.Cleanup)

	if err != nil {
		return err
	}
	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return err
	}

	manifestsForDeletion := map[string]service.Manifest{}

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
			manifestsForDeletion[manifestId] = manifest
		}
	}

	for manifestId, manifest := range manifestsForDeletion {
		for _, tag := range manifest.Tags {
			err = imagesService.Untag(tag)

			if err != nil {
				return err
			}

			fmt.Fprintf(writer, "Tag %s was removed from image. \n", tag)
		}

		err = imagesService.DeleteManifest(manifestId)

		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Image %s was removed.\n", manifestId)
	}

	return nil
}

func inArray(haystack []string, needle string) bool {
	for _, el := range haystack {
		if el == needle {
			return true
		}
	}

	return false
}

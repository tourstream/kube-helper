package registry

import (
	"io"
	"os"
	"strings"

	"kube-helper/loader"
	"kube-helper/service"
	"kube-helper/util"

	"github.com/urfave/cli"
	"fmt"
	"regexp"
)

var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var branchLoader loader.BranchLoaderInterface = new(loader.BranchLoader)
var imagesService service.ImagesInterface = new(service.Images)
var writer io.Writer = os.Stdout

func CmdCleanup(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	manifests, err := imagesService.List(configContainer.Cleanup)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	manifestsForDeletion := map[string]service.Manifest{}

	latestTagFound := false

	rp := regexp.MustCompile("staging-\\d")

	for _, manifestPair := range manifests.SortedManifests {
		cleanup := true
		for _, tag := range manifestPair.Value.Tags {

			if tag == "latest" {
				cleanup = false
				latestTagFound = true
				break

			}

			// only cleanup staging images which are older then the latest tag image
			if !latestTagFound && rp.MatchString(tag) {
				cleanup = false
			}

			if strings.HasPrefix(tag, "staging-") == false || tag == "staging-latest" {
				cleanup = false
				break
			}

			if strings.HasSuffix(tag, "latest") {

				branchName := strings.TrimSuffix(strings.TrimPrefix(tag, "staging-"), "-latest")

				//do not cleanup if branch exists
				if util.InArray(branches, branchName) {
					cleanup = false
					break
				}
			}
		}

		if cleanup {
			manifestsForDeletion[manifestPair.Key] = manifestPair.Value
		}
	}

	for manifestId, manifest := range manifestsForDeletion {
		for _, tag := range manifest.Tags {
			err = imagesService.Untag(configContainer.Cleanup, tag)

			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			fmt.Fprintf(writer, "Tag %s was removed from image. \n", tag)
		}

		err = imagesService.DeleteManifest(configContainer.Cleanup, manifestId)

		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		fmt.Fprintf(writer, "Image %s was removed.\n", manifestId)
	}

	return nil
}

package registry

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/mocks"
	"kube-helper/service"
)

func TestCmdCleanupWithWrongConfig(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(loader.Config{}, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanup(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		Cleanup: loader.Cleanup{
			ImagePath: "area.local/projectName/image-name",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldImagesLoader := imagesService
	imagesLoaderMock := new(mocks.ImagesInterface)

	imagesService = imagesLoaderMock

	collection := &service.TagCollection{
		Manifests: map[string]service.Manifest{
			"sha256:manifesthash": {
				Tags: []string{"tag-1", "tag-latest"},
			},
		},
	}

	imagesLoaderMock.On("List", config.Cleanup).Return(collection, nil)
	imagesLoaderMock.On("Untag", "tag-1").Return(nil)
	imagesLoaderMock.On("Untag", "tag-latest").Return(nil)
	imagesLoaderMock.On("DeleteManifest", "sha256:manifesthash").Return(nil)

	oldBranchLoader := branchLoader
	branchesLoaderMock := new(mocks.BranchLoaderInterface)

	branchLoader = branchesLoaderMock

	branchesLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"branch-1", "master"}, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		imagesService = oldImagesLoader
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	output := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, output, "Tag tag-1 was removed from image. \nTag tag-latest was removed from image. \nImage sha256:manifesthash was removed.\n")

}

func captureOutput(f func()) string {
	oldWriter := writer
	var buf bytes.Buffer
	defer func() { writer = oldWriter }()
	writer = &buf
	f()
	return buf.String()
}

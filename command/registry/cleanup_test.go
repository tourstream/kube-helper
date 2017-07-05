package registry

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/_mocks"
	"kube-helper/service"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdCleanupWithWrongConfig(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

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

func TestCmdCleanupWithErrorOnImageListCall(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		Cleanup: loader.Cleanup{
			ImagePath: "area.local/projectName/image-name",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldImagesLoader := imagesService
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesService = imagesLoaderMock

	imagesLoaderMock.On("List", config.Cleanup).Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		imagesService = oldImagesLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithErrorOnBranchesCall(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		Cleanup: loader.Cleanup{
			ImagePath: "area.local/projectName/image-name",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldImagesLoader := imagesService
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesService = imagesLoaderMock

	imagesLoaderMock.On("List", config.Cleanup).Return(&service.TagCollection{}, nil)

	branchesLoaderMock := new(_mocks.BranchLoaderInterface)

	oldBranchLoader := branchLoader
	branchLoader = branchesLoaderMock

	branchesLoaderMock.On("LoadBranches", config.Bitbucket).Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		imagesService = oldImagesLoader
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithErrorOnUntagCall(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		Cleanup: loader.Cleanup{
			ImagePath: "area.local/projectName/image-name",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldImagesLoader := imagesService
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesService = imagesLoaderMock

	collection := &service.TagCollection{
		Manifests: map[string]service.Manifest{
			"sha256:manifesthash2": {
				Tags: []string{"staging-a-s-s-s-s-1"},
			},
		},
	}

	imagesLoaderMock.On("List", config.Cleanup).Return(collection, nil)
	imagesLoaderMock.On("Untag", config.Cleanup, "staging-a-s-s-s-s-1").Return(errors.New("explode"))

	oldBranchLoader := branchLoader
	branchesLoaderMock := new(_mocks.BranchLoaderInterface)

	branchLoader = branchesLoaderMock

	branchesLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"branch-1", "master"}, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		imagesService = oldImagesLoader
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

}

func TestCmdCleanupWithErrorOnDeleteManifestCall(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		Cleanup: loader.Cleanup{
			ImagePath: "area.local/projectName/image-name",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldImagesLoader := imagesService
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesService = imagesLoaderMock

	collection := &service.TagCollection{
		Manifests: map[string]service.Manifest{
			"sha256:manifesthash2": {
				Tags: []string{"staging-a-s-s-s-s-1"},
			},
		},
	}

	imagesLoaderMock.On("List", config.Cleanup).Return(collection, nil)
	imagesLoaderMock.On("Untag", config.Cleanup, "staging-a-s-s-s-s-1").Return(nil)

	imagesLoaderMock.On("DeleteManifest", config.Cleanup, "sha256:manifesthash2").Return(errors.New("explode"))

	oldBranchLoader := branchLoader
	branchesLoaderMock := new(_mocks.BranchLoaderInterface)

	branchLoader = branchesLoaderMock

	branchesLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"branch-1", "master"}, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		imagesService = oldImagesLoader
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Contains(t, output, fmt.Sprintf("Tag %s was removed from image.", "staging-a-s-s-s-s-1"))

}

func TestCmdCleanupOnlyStaging(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		Cleanup: loader.Cleanup{
			ImagePath: "area.local/projectName/image-name",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldImagesLoader := imagesService
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesService = imagesLoaderMock

	collection := &service.TagCollection{
		Manifests: map[string]service.Manifest{
			"sha256:manifesthash": {
				Tags: []string{"tag-1", "tag-latest"},
			},
			"sha256:manifesthash2": {
				Tags: []string{"staging-a-s-s-s-s-1"},
			},
			"sha256:manifesthash3": {
				Tags: []string{"staging-a-s-s-s-s-2", "staging-tag-latest"},
			},
			"sha256:manifesthash4": {
				Tags: []string{"staging-branch-1-3"},
			},
			"sha256:manifesthash5": {
				Tags: []string{"staging-branch-1-4", "staging-branch-1-latest"},
			},
		},
	}

	expectedTags := []string{"staging-a-s-s-s-s-1", "staging-a-s-s-s-s-2", "staging-tag-latest", "staging-branch-1-3"}
	expectedManifests := []string{"sha256:manifesthash2", "sha256:manifesthash4", "sha256:manifesthash3"}

	imagesLoaderMock.On("List", config.Cleanup).Return(collection, nil)
	for _, expectedTag := range expectedTags {
		imagesLoaderMock.On("Untag", config.Cleanup, expectedTag).Return(nil)

	}
	for _, expectedManifest := range expectedManifests {
		imagesLoaderMock.On("DeleteManifest", config.Cleanup, expectedManifest).Return(nil)

	}

	oldBranchLoader := branchLoader
	branchesLoaderMock := new(_mocks.BranchLoaderInterface)

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

	for _, expectedTag := range expectedTags {
		assert.Contains(t, output, fmt.Sprintf("Tag %s was removed from image.", expectedTag))
	}

	for _, expectedManifest := range expectedManifests {
		assert.Contains(t, output, fmt.Sprintf("Image %s was removed.", expectedManifest))

	}
}

func captureOutput(f func()) string {
	oldWriter := writer
	var buf bytes.Buffer
	defer func() { writer = oldWriter }()
	writer = &buf
	f()
	return buf.String()
}

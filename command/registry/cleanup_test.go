package registry

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/_mocks"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"kube-helper/model"
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

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Empty(t, output)

	assert.Equal(t, "explode\n", errOutput)
}

func TestCmdCleanupWithErrorForImageService(t *testing.T) {
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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	serviceBuilderMock.On("GetImagesService").Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Empty(t, output)

	assert.Equal(t, "explode\n", errOutput)
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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	imagesLoaderMock := new(_mocks.ImagesInterface)

	serviceBuilder = serviceBuilderMock

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)


	imagesLoaderMock.On("List", config.Cleanup).Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Empty(t, output)

	assert.Equal(t, "explode\n", errOutput)
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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	imagesLoaderMock := new(_mocks.ImagesInterface)

	serviceBuilder = serviceBuilderMock

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)

	imagesLoaderMock.On("List", config.Cleanup).Return(&model.TagCollection{}, nil)

	branchesLoaderMock := new(_mocks.BranchLoaderInterface)

	oldBranchLoader := branchLoader
	branchLoader = branchesLoaderMock

	branchesLoaderMock.On("LoadBranches", config.Bitbucket).Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Empty(t, output)

	assert.Equal(t, "explode\n", errOutput)
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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	imagesLoaderMock := new(_mocks.ImagesInterface)

	serviceBuilder = serviceBuilderMock

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)

	collection := &model.TagCollection{
		SortedManifests: []model.ManifestPair{
			{
				Key: "sha256:manifesthash2",
				Value: model.Manifest{
					Tags: []string{"staging-a-s-s-s-s-1"},
				},
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
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Empty(t, output)

	assert.Equal(t, "explode\n", errOutput)

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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	imagesLoaderMock := new(_mocks.ImagesInterface)

	serviceBuilder = serviceBuilderMock

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)

	collection := &model.TagCollection{
		SortedManifests: []model.ManifestPair{
			{
				Key: "sha256:manifesthash2",
				Value: model.Manifest{
					Tags: []string{"staging-a-s-s-s-s-1"},
				},
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
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})
	assert.Equal(t, "explode\n", errOutput)
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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	imagesLoaderMock := new(_mocks.ImagesInterface)

	serviceBuilder = serviceBuilderMock

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)

	collection := &model.TagCollection{
		SortedManifests: []model.ManifestPair{
			{
				Key: "sha256:manifesthash",
				Value: model.Manifest{
					Tags: []string{"tag-1", "tag-latest"},
				},
			},
			{
				Key: "sha256:mainfest-staging-3",
				Value: model.Manifest{
					Tags: []string{"staging-31", "staging-latest"},
				},
			},
			{
				Key: "sha256:mainfest-staging-30",
				Value: model.Manifest{
					Tags: []string{"staging-30"},
				},
			},
			{
				Key: "sha256:mainfest-staging-28",
				Value: model.Manifest{
					Tags: []string{"staging-28", "latest"},
				},
			},
			{
				Key: "sha256:mainfest-staging-27",
				Value: model.Manifest{
					Tags: []string{"staging-27"},
				},
			},
			{
				Key: "sha256:manifesthash2",
				Value: model.Manifest{
					Tags: []string{"staging-a-s-s-s-s-1"},
				},
			},
			{
				Key: "sha256:manifesthash3",
				Value: model.Manifest{
					Tags: []string{"staging-a-s-s-s-s-2", "staging-tag-latest"},
				},
			},
			{
				Key: "sha256:manifesthash4",
				Value: model.Manifest{
					Tags: []string{"staging-branch-1-3"},
				},
			},
			{
				Key: "sha256:manifesthash5",
				Value: model.Manifest{
					Tags: []string{"staging-branch-1-4", "staging-branch-1-latest"},
				},
			},
		},
	}

	expectedTags := []string{"staging-27", "staging-a-s-s-s-s-1", "staging-a-s-s-s-s-2", "staging-tag-latest", "staging-branch-1-3"}
	expectedManifests := []string{"sha256:mainfest-staging-27", "sha256:manifesthash2", "sha256:manifesthash4", "sha256:manifesthash3"}

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
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Empty(t, errOutput)

	for _, expectedTag := range expectedTags {
		assert.Contains(t, output, fmt.Sprintf("Tag %s was removed from image.", expectedTag))
	}

	for _, expectedManifest := range expectedManifests {
		assert.Contains(t, output, fmt.Sprintf("Image %s was removed.", expectedManifest))

	}
}

func captureOutput(f func()) (string, string) {
	oldWriter := writer
	oldErrWriter := cli.ErrWriter
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	defer func() {
		writer = oldWriter
		cli.ErrWriter = oldErrWriter
	}()
	writer = &buf
	cli.ErrWriter = &errBuf
	f()
	return buf.String(), errBuf.String()
}

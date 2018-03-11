package app

import (
	"errors"
	"testing"

	"kube-helper/_mocks"
	"kube-helper/command"
	"kube-helper/loader"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdApplyWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdApply, []string{"apply", "-c", "never.yml", "foobar"})
}

func TestCmdApplyWithErrorForApplicationService(t *testing.T) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	oldApplicationServiceCreator := applicationServiceCreator
	serviceBuilderMock := new(_mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdApply, []string{"apply", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdApplyWithErrorForImagesService(t *testing.T) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	oldApplicationServiceCreator := applicationServiceCreator

	serviceBuilderMock := new(_mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, new(_mocks.ApplicationServiceInterface), nil)

	serviceBuilderMock.On("GetImagesService").Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdApply, []string{"apply", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdApplyWithErrorForHasTag(t *testing.T) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	oldApplicationServiceCreator := applicationServiceCreator

	serviceBuilderMock := new(_mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeApplicationService := new(_mocks.ApplicationServiceInterface)
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-latest").Return(false, errors.New("explode"))

	applicationServiceCreator = mockNewApplicationService(t, "staging", config, fakeApplicationService, nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdApply, []string{"apply", "-c", "never.yml", "master"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdApplyWithFalseForHasTag(t *testing.T) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	oldApplicationServiceCreator := applicationServiceCreator
	serviceBuilderMock := new(_mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeApplicationService := new(_mocks.ApplicationServiceInterface)
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesLoaderMock.On("HasTag", config.Cleanup, "latest").Return(false, nil)

	applicationServiceCreator = mockNewApplicationService(t, "production", config, fakeApplicationService, nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdApply, []string{"apply", "-c", "never.yml", "-p"})
	})

	assert.Equal(t, "No Image 'latest' found for namespace 'production' \n", errOutput)
	assert.Empty(t, output)
}

func TestCmdApplyWithErrorForCreateApplication(t *testing.T) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	oldApplicationServiceCreator := applicationServiceCreator
	serviceBuilderMock := new(_mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeApplicationService := new(_mocks.ApplicationServiceInterface)
	imagesLoaderMock := new(_mocks.ImagesInterface)

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, fakeApplicationService, nil)

	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-foobar-latest").Return(true, nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)

	fakeApplicationService.On("Apply").Return(errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdApply, []string{"apply", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdApply(t *testing.T) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock
	oldApplicationServiceCreator := applicationServiceCreator

	fakeApplicationService := new(_mocks.ApplicationServiceInterface)

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, fakeApplicationService, nil)

	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-foobar-latest").Return(true, nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesLoaderMock, nil)

	fakeApplicationService.On("Apply").Return(nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdApply, []string{"apply", "-c", "never.yml", "foobar"})
	})

	assert.Empty(t, errOutput)
	assert.Empty(t, output)
}

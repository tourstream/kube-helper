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

func TestCmdShutdownWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdShutdown, []string{"shutdown", "-c", "never.yml", "foobar"})
}

func TestCmdShutdownWithErrorForApplicationService(t *testing.T) {

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

	oldApplicationServiceCreator := applicationServiceCreator

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdShutdown, []string{"shutdown", "-c", "never.yml", "foobar"})
	})

	assert.Empty(t, output)
	assert.Equal(t, "explode\n", errOutput)
}

func TestCmdShutdownWithErrorForDeleteNamespace(t *testing.T) {

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

	oldApplicationServiceCreator := applicationServiceCreator

	fakeApplicationService := new(_mocks.ApplicationServiceInterface)

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, fakeApplicationService, nil)

	fakeApplicationService.On("DeleteByNamespace").Return(errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdShutdown, []string{"shutdown", "-c", "never.yml", "foobar"})
	})

	assert.Empty(t, output)
	assert.Equal(t, "explode\n", errOutput)
}

func TestCmdShutdown(t *testing.T) {

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

	oldApplicationServiceCreator := applicationServiceCreator

	fakeApplicationService := new(_mocks.ApplicationServiceInterface)

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, fakeApplicationService, nil)

	fakeApplicationService.On("DeleteByNamespace").Return(nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdShutdown, []string{"shutdown", "-c", "never.yml", "foobar"})
	})

	assert.Empty(t, output)
	assert.Empty(t, errOutput)
}

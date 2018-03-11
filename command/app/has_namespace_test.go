package app

import (
	"errors"
	"testing"

	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdHasNamespaceWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})
}

func TestCmdHasNamespaceWithErrorForGetApplicationService(t *testing.T) {
	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldApplicationServiceCreator := applicationServiceCreator

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, nil, errors.New("explode"))

	oldHandler := cli.OsExiter
	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	defer func() {
		applicationServiceCreator = oldApplicationServiceCreator
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
	}()

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)

}

func TestCmdHasNamespaceShouldReturnFalseIfNameSpaceNotFound(t *testing.T) {
	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldApplicationServiceCreator := applicationServiceCreator

	appService := new(mocks.ApplicationServiceInterface)

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, appService, nil)

	appService.On("HasNamespace").Return(false)

	oldHandler := cli.OsExiter
	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	defer func() {
		applicationServiceCreator = oldApplicationServiceCreator
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
	}()

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})
	})

	assert.Empty(t, errOutput)
	assert.Equal(t, output, "false")

}

func TestCmdHasNamespaceShouldReturnTrueIfNameSpaceFound(t *testing.T) {

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldApplicationServiceCreator := applicationServiceCreator

	appService := new(mocks.ApplicationServiceInterface)

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, appService, nil)

	appService.On("HasNamespace").Return(true)

	oldHandler := cli.OsExiter
	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	defer func() {
		applicationServiceCreator = oldApplicationServiceCreator
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
	}()

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})
	})

	assert.Empty(t, errOutput)
	assert.Equal(t, output, "true")
}

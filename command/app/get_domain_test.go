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

func TestCmdGetDomainWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdGetDomain, []string{"get-domain", "-c", "never.yml", "foobar"})
}

func TestCmdGetDomainWithErrorForGetApplicationService(t *testing.T) {
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
		command.RunTestCommand(CmdGetDomain, []string{"get-domain", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, errOutput, "explode\n")
	assert.Empty(t, output)

}

func TestCmdGetDomain(t *testing.T) {

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	appService := new(mocks.ApplicationServiceInterface)

	appService.On("GetDomain", loader.DNSConfig{}).Return("domain")

	oldApplicationServiceCreator := applicationServiceCreator

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, appService, nil)

	oldHandler := cli.OsExiter
	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdGetDomain, []string{"get-domain", "-c", "never.yml", "foobar"})
	})

	assert.Empty(t, errOutput)
	assert.Equal(t, output, "domain")
}

package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/client-go/kubernetes/fake"
	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/_mocks"
)

func TestCmdGetDomainWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdGetDomain, []string{"get-domain", "-c", "never.yml", "foobar"})
}

func TestCmdGetDomainWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdGetDomain, []string{"get-domain", "-c", "never.yml", "foorbar"})
}

func TestCmdGetDomainWithErrorForGetApplicationService(t *testing.T) {
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
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := fake.NewSimpleClientset()

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(nil, errors.New("explode"))

	oldHandler := cli.OsExiter
	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}



	defer func() {
		serviceBuilder = oldServiceBuilder
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
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := fake.NewSimpleClientset()
	appService := new(_mocks.ApplicationServiceInterface)

	appService.On("GetDomain", loader.DNSConfig{}).Return("domain")

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(appService, nil)

	oldHandler := cli.OsExiter
	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	defer func() {
		serviceBuilder = oldServiceBuilder
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
	}()

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdGetDomain, []string{"get-domain", "-c", "never.yml", "foobar"})
	})

	assert.Empty(t, errOutput)
	assert.Equal(t, output, "domain")
}

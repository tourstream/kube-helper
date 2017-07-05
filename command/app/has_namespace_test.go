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

func TestCmdHasNamespaceWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})
}

func TestCmdHasNamespaceWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdHasNamespace, []string{"has-namesapce", "-c", "never.yml", "foorbar"})
}

func TestCmdHasNamespaceWithErrorForGetApplicationService(t *testing.T) {
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

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)
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

	output := captureErrorOutput(func() {
		command.RunTestCommand(CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, output, "explode\n")

}

func TestCmdHasNamespaceShouldReturnFalseIfNameSpaceNotFound(t *testing.T) {
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

	appService.On("HasNamespace").Return(false)

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)
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

	output := captureOutput(func() {
		command.RunTestCommand(CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, output, "false")

}

func TestCmdHasNamespaceShouldReturnTrueIfNameSpaceFound(t *testing.T) {

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

	appService.On("HasNamespace").Return(true)

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)
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

	output := captureOutput(func() {
		command.RunTestCommand(CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, output, "true")
}

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

func TestCmdShutdownWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdShutdown, []string{"shutdown", "-c", "never.yml", "foobar"})
}

func TestCmdShutdownWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdShutdown, []string{"shutdown", "-c", "never.yml", "foorbar"})
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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := new(fake.Clientset)

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output := captureErrorOutput(func() {
		command.RunTestCommand(CmdShutdown, []string{"shutdown", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, "explode\n", output)
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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := new(fake.Clientset)
	fakeApplicationService := new(_mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(fakeApplicationService, nil)

	fakeApplicationService.On("DeleteByNamespace").Return(errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output := captureErrorOutput(func() {
		command.RunTestCommand(CmdShutdown, []string{"shutdown", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, "explode\n", output)
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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := new(fake.Clientset)
	fakeApplicationService := new(_mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(fakeApplicationService, nil)

	fakeApplicationService.On("DeleteByNamespace").Return(nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}
	command.RunTestCommand(CmdShutdown, []string{"shutdown", "-c", "never.yml", "foobar"})
}

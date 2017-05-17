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

func TestCmdUpdateWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdUpdate, []string{"update", "-c", "never.yml", "foobar"})
}

func TestCmdUpdateWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdUpdate, []string{"update", "-c", "never.yml", "foorbar"})
}

func TestCmdUpdateWithErrorForApplicationService(t *testing.T) {

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
		command.RunTestCommand(CmdUpdate, []string{"update", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, "explode\n", output)
}

func TestCmdUpdateWithErrorForCreateApplication(t *testing.T) {

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

	fakeApplicationService.On("UpdateByNamespace").Return(errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output := captureErrorOutput(func() {
		command.RunTestCommand(CmdUpdate, []string{"update", "-c", "never.yml", "foobar"})
	})

	assert.Equal(t, "explode\n", output)
}

func TestCmdUpdate(t *testing.T) {

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

	fakeApplicationService.On("UpdateByNamespace").Return(nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}
	command.RunTestCommand(CmdUpdate, []string{"update", "-c", "never.yml", "foobar"})
}
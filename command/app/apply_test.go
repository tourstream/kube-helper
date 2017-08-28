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

func TestCmdApplyWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdApply, []string{"apply", "-c", "never.yml", "foobar"})
}

func TestCmdApplyWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdApply, []string{"apply", "-c", "never.yml", "foorbar"})
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
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := new(fake.Clientset)

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
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
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := new(fake.Clientset)
	fakeApplicationService := new(_mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(fakeApplicationService, nil)

	fakeApplicationService.On("Apply").Return(errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
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
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := new(fake.Clientset)
	fakeApplicationService := new(_mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(fakeApplicationService, nil)

	fakeApplicationService.On("Apply").Return(nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
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

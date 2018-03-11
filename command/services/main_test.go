package services

import (
	"bytes"
	"errors"
	"testing"

	"kube-helper/_mocks"
	"kube-helper/command"
	"kube-helper/loader"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func helperTestCmdHasWrongConfigReturned(t *testing.T, Action interface{}, arguments []string) {

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
		command.RunTestCommand(Action, arguments)
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)

}

func helperTestCmdlWithErrorForClientSet(t *testing.T, Action interface{}, arguments []string) {

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

	serviceBuilderMock.On("GetClientSet", config).Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(Action, arguments)
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)

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

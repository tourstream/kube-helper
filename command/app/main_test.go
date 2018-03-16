package app

import (
	"bytes"
	"errors"
	"testing"

	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/mocks"

	"kube-helper/service/app"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetNamespace(t *testing.T) {
	for _, name := range []string{"", "master", "staging"} {
		assert.Equal(t, getNamespace(name, false), "staging")
	}

	for _, name := range []string{"", "master", "staging"} {
		assert.Equal(t, getNamespace(name, true), "production")
	}
}

func helperTestCmdHasWrongConfigReturned(t *testing.T, Action interface{}, arguments []string) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoader)

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
	configLoaderMock := new(mocks.ConfigLoader)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(mocks.ServiceBuilderInterface)

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

func testNamespace(ns string) v1.Namespace {
	return v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: ns,
		},
	}
}

func mockNewApplicationService(t *testing.T, expectedNamespace string, expectedConfig loader.Config, serviceMock app.ApplicationServiceInterface, err error) func(namespace string, config loader.Config) (app.ApplicationServiceInterface, error) {
	return func(namespace string, config loader.Config) (app.ApplicationServiceInterface, error) {
		assert.Equal(t, expectedConfig, config)
		assert.Equal(t, expectedNamespace, namespace)

		return serviceMock, err
	}
}

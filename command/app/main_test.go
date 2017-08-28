package app

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/client-go/pkg/api/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kube-helper/_mocks"
	"kube-helper/command"
	"kube-helper/loader"
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
	serviceBuilderMock := new(_mocks.BuilderInterface)

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

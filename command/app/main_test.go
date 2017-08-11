package app

import (
	"bytes"
	"errors"
	"log"
	"os"
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

	output := captureErrorOutput(func() {
		command.RunTestCommand(Action, arguments)
	})

	assert.Equal(t, "explode\n", output)

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

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output := captureErrorOutput(func() {
		command.RunTestCommand(Action, arguments)
	})

	assert.Equal(t, "explode\n", output)

}

func captureErrorOutput(f func()) string {
	oldWriter := cli.ErrWriter
	var buf bytes.Buffer
	defer func() { cli.ErrWriter = oldWriter }()
	cli.ErrWriter = &buf
	f()
	return buf.String()
}

func captureOutput(f func()) string {
	oldWriter := writer
	var buf bytes.Buffer
	defer func() { writer = oldWriter }()
	writer = &buf
	f()
	return buf.String()
}

func captureLogOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stderr)
	return buf.String()
}

func testNamespace(ns string) v1.Namespace {
	return v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: ns,
		},
	}
}

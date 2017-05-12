package app

import (
	"bytes"
	"log"
	"os"
	"testing"

	"errors"
	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
)

func TestCmdHasNamespaceWithWrongConf(t *testing.T) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(loader.Config{}, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	command.RunTestCommand(CmdHasNamespace, []string{"has-namespace", "-c", "never.yml", "foobar"})


}

func TestCmdHasNamespaceShouldReturnFalseIfNameSpaceNotFound(t *testing.T) {
	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := fake.NewSimpleClientset()

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)

	oldHandler := cli.OsExiter
	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

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

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	namespaceFake := new(v1.Namespace)
	namespaceFake.Name = "foobar"
	fakeClientSet := fake.NewSimpleClientset(namespaceFake)

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)

	oldHandler := cli.OsExiter
	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

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

func captureLogOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stderr)
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

package services

import (
	"errors"
	"testing"

	"kube-helper/_mocks"
	"kube-helper/command"
	"kube-helper/loader"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	testing_k8s "k8s.io/client-go/testing"
)

func TestCmdGetIpWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdGetIp, []string{"get-ip", "-c", "never.yml", "foobar", "dummy"})
}

func TestCmdGetIpWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdGetIp, []string{"get-ip", "-c", "never.yml", "foobar", "dummy"})
}

func TestCmdGetIpWithErrorForGetService(t *testing.T) {

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
	fakeClientSet.PrependReactor("get", "services", errorReturnFunc)

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdGetIp, []string{"get-ip", "-c", "never.yml", "foobar", "dummy"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdGetIp(t *testing.T) {

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

	serviceMock := &v1.Service{
		Spec: v1.ServiceSpec{
			ClusterIP: "127.0.0.1",
		},
	}

	fakeClientSet := new(fake.Clientset)
	fakeClientSet.PrependReactor("get", "services", getObjectReturnFunc(serviceMock))

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdGetIp, []string{"get-ip", "-c", "never.yml", "foobar", "dummy"})
	})

	assert.Equal(t, "127.0.0.1\n", output)
	assert.Empty(t, errOutput)
}

func errorReturnFunc(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

	return true, nil, errors.New("explode")
}

func nilReturnFunc(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

	return true, nil, nil
}

func getObjectReturnFunc(obj runtime.Object) testing_k8s.ReactionFunc {
	return func(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

		return true, obj, nil
	}
}

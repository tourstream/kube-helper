package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	testing2 "k8s.io/client-go/testing"
	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/_mocks"
)

func TestCmdShutdownAllWithWrongConf(t *testing.T) {

	helperTestCmdHasWrongConfigReturned(t, CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
}

func TestCmdShutdownAllWithErrorForClientSet(t *testing.T) {

	helperTestCmdlWithErrorForClientSet(t, CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
}

func TestCmdShutdownAllWithErrorForGetNamespaceList(t *testing.T) {

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

	fakeClientSet.AddReactor("list", "*", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("explode")
	})

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output := captureErrorOutput(func() {
		command.RunTestCommand(CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", output)
}

func TestCmdShutdownAllWithErrorGetApplicationService(t *testing.T) {

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

	namespaceList := &v1.NamespaceList{
		Items: []v1.Namespace{testNamespace("default"), testNamespace("kube-system"), testNamespace("foobar")},
	}

	fakeClientSet := fake.NewSimpleClientset(namespaceList)

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
		command.RunTestCommand(CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", output)
}

func TestCmdShutdownAllWithErrorDeleteNamespace(t *testing.T) {

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

	namespaceList := &v1.NamespaceList{
		Items: []v1.Namespace{testNamespace("default"), testNamespace("kube-system"), testNamespace("foobar")},
	}

	fakeClientSet := fake.NewSimpleClientset(namespaceList)
	appService := new(_mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", "test-project", "berlin", "testing").Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(appService, nil)

	appService.On("DeleteByNamespace").Return(errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	output := captureOutput(func() {
		command.RunTestCommand(CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
	})

	assert.Equal(t, "explode", output)
}



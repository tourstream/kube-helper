package app

import (
	"errors"
	"testing"

	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	testing2 "k8s.io/client-go/testing"
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

	fakeClientSet := new(fake.Clientset)

	fakeClientSet.AddReactor("list", "*", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("explode")
	})

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
		command.RunTestCommand(CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdShutdownAllWithErrorGetApplicationService(t *testing.T) {

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
	oldApplicationServiceCreator := applicationServiceCreator
	serviceBuilderMock := new(mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock

	namespaceList := &v1.NamespaceList{
		Items: []v1.Namespace{testNamespace("default"), testNamespace("kube-system"), testNamespace("foobar")},
	}

	fakeClientSet := fake.NewSimpleClientset(namespaceList)

	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, nil, errors.New("explode"))

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdShutdownAllWithErrorDeleteNamespace(t *testing.T) {

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
	oldApplicationServiceCreator := applicationServiceCreator
	serviceBuilderMock := new(mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock

	namespaceList := &v1.NamespaceList{
		Items: []v1.Namespace{testNamespace("default"), testNamespace("kube-system"), testNamespace("foobar")},
	}

	fakeClientSet := fake.NewSimpleClientset(namespaceList)
	appService := new(mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, appService, nil)

	appService.On("DeleteByNamespace").Return(errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
	})

	assert.Equal(t, "explode", output)
	assert.Empty(t, errOutput)
}

func TestCmdShutdownAllWithErrorDeleteNamespaceWithPrefix(t *testing.T) {

	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoader)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
		Namespace: loader.Namespace{
			Prefix: "dummy",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	oldApplicationServiceCreator := applicationServiceCreator
	serviceBuilderMock := new(mocks.ServiceBuilderInterface)

	serviceBuilder = serviceBuilderMock

	namespaceList := &v1.NamespaceList{
		Items: []v1.Namespace{testNamespace("default"), testNamespace("kube-system"), testNamespace("dummy-foobar")},
	}

	fakeClientSet := fake.NewSimpleClientset(namespaceList)
	appService := new(mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	applicationServiceCreator = mockNewApplicationService(t, "foobar", config, appService, nil)

	appService.On("DeleteByNamespace").Return(errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		applicationServiceCreator = oldApplicationServiceCreator
	}()

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdShutdownAll, []string{"shutdown-all", "-c", "never.yml"})
	})

	assert.Equal(t, "explode", output)
	assert.Empty(t, errOutput)
}

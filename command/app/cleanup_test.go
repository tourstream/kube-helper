package app

import (
	"errors"
	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/_mocks"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	testing2 "k8s.io/client-go/testing"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCmdCleanupWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdCleanUp, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdCleanUp, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithErrorForGetBranches(t *testing.T) {
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
	serviceBuilderMock.On("GetClientSet", config).Return(fake.NewSimpleClientset(), nil)

	serviceBuilder = serviceBuilderMock

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return(nil, errors.New("explode"))

	branchLoader = branchLoaderMock

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		branchLoader = oldBranchLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanUp, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdCleanupWithErrorForGetNamespaces(t *testing.T) {
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

	fakeClientSet := new(fake.Clientset)

	fakeClientSet.AddReactor("list", "*", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("explode")
	})

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)

	serviceBuilder = serviceBuilderMock

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{}, nil)

	branchLoader = branchLoaderMock

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		branchLoader = oldBranchLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanUp, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdCleanupWithErrorForInitApplicationService(t *testing.T) {
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

	namespaceList := &v1.NamespaceList{
		Items: []v1.Namespace{testNamespace("default"), testNamespace("kube-system"), testNamespace("foobar")},
	}

	fakeClientSet := fake.NewSimpleClientset(namespaceList)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(nil, errors.New("explode"))

	serviceBuilder = serviceBuilderMock

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"test"}, nil)

	branchLoader = branchLoaderMock

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		branchLoader = oldBranchLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanUp, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdCleanupWithErrorForDeleteNamespace(t *testing.T) {
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

	namespaceList := &v1.NamespaceList{
		Items: []v1.Namespace{testNamespace("default"), testNamespace("kube-system"), testNamespace("foobar")},
	}

	fakeClientSet := fake.NewSimpleClientset(namespaceList)
	appService := new(_mocks.ApplicationServiceInterface)

	appService.On("DeleteByNamespace").Return(errors.New("explode"))

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "foobar", config).Return(appService, nil)

	serviceBuilder = serviceBuilderMock

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"test"}, nil)

	branchLoader = branchLoaderMock

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		branchLoader = oldBranchLoader
		serviceBuilder = oldServiceBuilder
	}()

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanUp, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "explode", output)
	assert.Empty(t, errOutput)
}

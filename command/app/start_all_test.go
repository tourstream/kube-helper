package app

import (
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"k8s.io/client-go/kubernetes/fake"
	"kube-helper/_mocks"
	"kube-helper/command"
	"kube-helper/loader"
)

func TestCmdStartUpAllWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
}

func TestCmdStartUpAllWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
}

func TestCmdStartUpAllWithErrorForCopyEnv(t *testing.T) {

	appFS := afero.NewMemMapFs()

	oldHandler := cli.OsExiter
	oldFileSystem := fileSystem

	fileSystem = appFS

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

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		fileSystem = oldFileSystem
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
	})

	assert.Equal(t, "open .env: file does not exist\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdStartUpAllWithErrorForRemoveEnv(t *testing.T) {

	appFS := afero.NewMemMapFs()

	afero.WriteFile(appFS, ".env", []byte("key: ###FOO###\n---\ntest: ###FOOBAR###"), 0444)

	oldHandler := cli.OsExiter
	oldFileSystem := fileSystem

	fileSystem = afero.NewReadOnlyFs(appFS)

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

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		fileSystem = oldFileSystem
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
	})

	assert.Empty(t, output)
	assert.Equal(t, "operation not permitted\n", errOutput)
}

func TestCmdStartUpAllWithErrorForLoadBranches(t *testing.T) {

	appFS := afero.NewMemMapFs()

	afero.WriteFile(appFS, ".env", []byte("key: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	oldHandler := cli.OsExiter
	oldFileSystem := fileSystem

	fileSystem = appFS

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

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return(nil, errors.New("explode"))

	branchLoader = branchLoaderMock

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		fileSystem = oldFileSystem
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
	})

	assert.Empty(t, output)
	assert.Equal(t, "explode\n", errOutput)
}

func TestCmdStartUpAllWithoutDataBase(t *testing.T) {

	appFS := afero.NewMemMapFs()

	afero.WriteFile(appFS, ".env", []byte("key: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	oldHandler := cli.OsExiter
	oldFileSystem := fileSystem

	fileSystem = appFS

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
		Cleanup: loader.Cleanup{
			ImagePath: "eu.gcr.io/noop/assad",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := new(fake.Clientset)
	fakeAppService := new(_mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "branch-3", config).Return(fakeAppService, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "staging", config).Return(fakeAppService, nil)

	fakeAppService.On("Apply").Return(nil)

	oldBranchLoader := branchLoader
	branchesLoaderMock := new(_mocks.BranchLoaderInterface)

	branchesLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"branch-3", "branch-2", "branch-1", "master"}, nil)

	branchLoader = branchesLoaderMock

	oldImagesLoader := imagesService
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesService = imagesLoaderMock

	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-branch-2-latest").Return(false, errors.New("explode"))
	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-branch-1-latest").Return(false, nil)
	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-branch-3-latest").Return(true, nil)
	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-latest").Return(true, nil)

	oldDatabaseCopy := databaseCopy
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function

	databaseCopy = func(branchname string, config loader.Config) error {
		return nil

	}

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		fileSystem = oldFileSystem
		branchLoader = oldBranchLoader
		imagesService = oldImagesLoader
		databaseCopy = oldDatabaseCopy
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
	})

	assert.Empty(t, errOutput)

	bytes, err := afero.ReadFile(appFS, ".env")

	stringDat := string(bytes)

	assert.NoError(t, err)
	assert.Equal(t, "key: ###FOO###\n---\ntest: ###FOOBAR###", stringDat)

	assert.Equal(t, "explode\n", output)
}

func TestCmdStartUpAll(t *testing.T) {

	appFS := afero.NewMemMapFs()

	afero.WriteFile(appFS, ".env", []byte("key: ###FOO###\n---\ntest: ###FOOBAR###"), 0644)

	oldHandler := cli.OsExiter
	oldFileSystem := fileSystem

	fileSystem = appFS

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	config := loader.Config{
		ProjectID: "test-project",
		Zone:      "berlin",
		ClusterID: "testing",
		Cleanup: loader.Cleanup{
			ImagePath: "eu.gcr.io/noop/assad",
		},
		Database: loader.Database{
			Instance: "dummy",
		},
	}

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)

	serviceBuilder = serviceBuilderMock

	fakeClientSet := new(fake.Clientset)
	fakeAppService := new(_mocks.ApplicationServiceInterface)

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "branch-3", config).Return(fakeAppService, nil)
	serviceBuilderMock.On("GetApplicationService", fakeClientSet, "staging", config).Return(fakeAppService, nil)

	fakeAppService.On("Apply").Return(nil)

	oldBranchLoader := branchLoader
	branchesLoaderMock := new(_mocks.BranchLoaderInterface)

	branchesLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"branch-3", "branch-2", "branch-1", "master"}, nil)

	branchLoader = branchesLoaderMock

	oldImagesLoader := imagesService
	imagesLoaderMock := new(_mocks.ImagesInterface)

	imagesService = imagesLoaderMock

	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-branch-2-latest").Return(false, errors.New("explode"))
	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-branch-1-latest").Return(false, nil)
	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-branch-3-latest").Return(true, nil)
	imagesLoaderMock.On("HasTag", config.Cleanup, "staging-latest").Return(true, nil)

	oldDatabaseCopy := databaseCopy
	// as we are exiting, revert sqlOpen back to oldSqlOpen at end of function

	databaseCopy = func(branchname string, config loader.Config) error {
		return nil

	}

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		fileSystem = oldFileSystem
		branchLoader = oldBranchLoader
		imagesService = oldImagesLoader
		databaseCopy = oldDatabaseCopy
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
	})

	assert.Empty(t, errOutput)
	assert.Equal(t, "explode\n", output)
}

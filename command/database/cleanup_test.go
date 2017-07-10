package database

import (
	"errors"
	"testing"

	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/_mocks"

	"bytes"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"gopkg.in/h2non/gock.v1"
)

func TestCmdCleanupWithWrongConfig(t *testing.T) {
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

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithWrongSqlService(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(loader.Config{}, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilder = serviceBuilderMock

	serviceBuilderMock.On("GetSqlService").Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithFailureLoadBranches(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{}

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilder = serviceBuilderMock

	sqlService, err := oldServiceBuilder.GetSqlService()
	serviceBuilderMock.On("GetSqlService").Return(sqlService, err)

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)
	branchLoader = branchLoaderMock

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return(nil, errors.New("explode"))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithFailureLoadDatabases(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		ProjectID: "test-project",
		Database: loader.Database{
			Instance: "testing",
		},
	}

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilder = serviceBuilderMock

	sqlService, err := oldServiceBuilder.GetSqlService()
	serviceBuilderMock.On("GetSqlService").Return(sqlService, err)

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)
	branchLoader = branchLoaderMock

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"master", "ets-123"}, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/instances/testing/databases").
		Reply(404).
		JSON(map[string]string{"foo": "bar"})

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithFailureForDelete(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		ProjectID: "test-project",
		Database: loader.Database{
			Instance: "testing",
		},
	}

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilder = serviceBuilderMock

	sqlService, err := oldServiceBuilder.GetSqlService()
	serviceBuilderMock.On("GetSqlService").Return(sqlService, err)

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)
	branchLoader = branchLoaderMock

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"master", "ets-123"}, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	listResponse := `
	{
	"kind": "sql#databasesList",
		"items": [
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/n2170-container-engine-spike/instances/staging/databases/information_schema",
	"name": "information_schema",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/icuaZXnvrCc7-1kwIC7BwTFfgf0\"",
	"project": "n2170-container-engine-spike",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	},
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/n2170-container-engine-spike/instances/staging/databases/landing_page_generator",
	"name": "landing_page_generator",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/TOxfaosOQfEN34mlC3_6aizAB4Q\"",
	"project": "n2170-container-engine-spike",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	}]}
	`

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/instances/testing/databases").
		Reply(200).
		JSON(listResponse)

	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	gock.New("https://www.googleapis.com").
		Delete("/sql/v1beta4/projects/test-project/instances/testing/databases/landing_page_generator").
		Reply(404).
		JSON(map[string]string{"foo": "bar"})

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanupWithFailureDuringWait(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		ProjectID: "test-project",
		Database: loader.Database{
			Instance: "testing",
		},
	}

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilder = serviceBuilderMock

	sqlService, err := oldServiceBuilder.GetSqlService()
	serviceBuilderMock.On("GetSqlService").Return(sqlService, err)

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)
	branchLoader = branchLoaderMock

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"master", "ets-123"}, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	defer gock.Off() // Flush pending mocks after test execution

	for i := 0; i < 3; i++ {
		gock.New("https://accounts.google.com").
			Post("/o/oauth2/token").
			Reply(200).
			JSON(map[string]string{"foo": "bar"})
	}

	listResponse := `
	{
	"kind": "sql#databasesList",
		"items": [
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/n2170-container-engine-spike/instances/staging/databases/information_schema",
	"name": "ets-123",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/icuaZXnvrCc7-1kwIC7BwTFfgf0\"",
	"project": "n2170-container-engine-spike",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	},
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/n2170-container-engine-spike/instances/staging/databases/landing_page_generator",
	"name": "landing_page_generator",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/TOxfaosOQfEN34mlC3_6aizAB4Q\"",
	"project": "n2170-container-engine-spike",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	}]}
	`

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/instances/testing/databases").
		Reply(200).
		JSON(listResponse)

	operationResponse := `{
		"kind": "sql#operation",
		"status" : "PENDING",
		"name": "uuid"
		}
	`

	gock.New("https://www.googleapis.com").
		Delete("/sql/v1beta4/projects/test-project/instances/testing/databases/landing_page_generator").
		Reply(200).
		JSON(operationResponse)

	operationResponse = `{
		"kind": "sql#operation",
		"status" : "DONE",
		"name": "uuid",
		"error": {
		    "kind": "sql#operationErrors",
		    "errors": [
		      {
			"kind": "sql#operationError",
			"code": "201",
			"message": "explode"
		      }
		    ]
		  }
		}`

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/operations/uuid").
		Reply(200).
		JSON(operationResponse)

	command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
}

func TestCmdCleanup(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		ProjectID: "test-project",
		Database: loader.Database{
			Instance: "testing",
			BaseName: "base",
		},
	}

	oldConfigLoader := configLoader
	configLoaderMock := new(_mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(_mocks.BuilderInterface)
	serviceBuilder = serviceBuilderMock

	sqlService, err := oldServiceBuilder.GetSqlService()
	serviceBuilderMock.On("GetSqlService").Return(sqlService, err)

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)
	branchLoader = branchLoaderMock

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"master", "ets-123"}, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 0, exitCode)
	}

	defer gock.Off() // Flush pending mocks after test execution

	for i := 0; i < 2; i++ {
		gock.New("https://accounts.google.com").
			Post("/o/oauth2/token").
			Reply(200).
			JSON(map[string]string{"foo": "bar"})
	}

	listResponse := `
	{
	"kind": "sql#databasesList",
		"items": [
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/n2170-container-engine-spike/instances/staging/databases/information_schema",
	"name": "base",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/icuaZXnvrCc7-1kwIC7BwTFfgf0\"",
	"project": "n2170-container-engine-spike",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	},
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/n2170-container-engine-spike/instances/staging/databases/landing_page_generator",
	"name": "landing_page_generator",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/TOxfaosOQfEN34mlC3_6aizAB4Q\"",
	"project": "n2170-container-engine-spike",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	}]}
	`

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/instances/testing/databases").
		Reply(200).
		JSON(listResponse)

	operationResponse := `{
		"kind": "sql#operation",
		"status" : "DONE",
		"name": "uuid"
		}
	`

	gock.New("https://www.googleapis.com").
		Delete("/sql/v1beta4/projects/test-project/instances/testing/databases/landing_page_generator").
		Reply(200).
		JSON(operationResponse)
	output := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "Removed database landing_page_generator", output)
}

func captureOutput(f func()) string {
	oldWriter := writer
	var buf bytes.Buffer
	defer func() { writer = oldWriter }()
	writer = &buf
	f()
	return buf.String()
}

func captureErrorOutput(f func()) string {
	oldWriter := cli.ErrWriter
	var buf bytes.Buffer
	defer func() { cli.ErrWriter = oldWriter }()
	cli.ErrWriter = &buf
	f()
	return buf.String()
}

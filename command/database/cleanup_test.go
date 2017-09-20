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
	"time"
	util_clock "k8s.io/apimachinery/pkg/util/clock"
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

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
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

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
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

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "explode\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdCleanupWithFailureLoadDatabases(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		Cluster: loader.Cluster{
			ProjectID: "test-project",
		},
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

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "googleapi: got HTTP response code 404 with body: {\"foo\":\"bar\"}\n\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdCleanupWithFailureForDelete(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		Cluster: loader.Cluster{
			ProjectID: "test-project",
		},
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
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/staging/databases/information_schema",
	"name": "information_schema",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/icuaZXnvrCc7-1kwIC7BwTFfgf0\"",
	"project": "test-project",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	},
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/staging/databases/foobar_database",
	"name": "foobar_database",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/TOxfaosOQfEN34mlC3_6aizAB4Q\"",
	"project": "test-project",
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
		Delete("/sql/v1beta4/projects/test-project/instances/testing/databases/foobar_database").
		Reply(404).
		JSON(map[string]string{"foo": "bar"})

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "googleapi: got HTTP response code 404 with body: {\"foo\":\"bar\"}\n\n", errOutput)
	assert.Empty(t, output)
}

func TestCmdCleanupWithFailureDuringWait(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		Cluster: loader.Cluster{
			ProjectID: "test-project",
		},
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

	oldClock := clock
	clock = util_clock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

	oldBranchLoader := branchLoader
	branchLoaderMock := new(_mocks.BranchLoaderInterface)
	branchLoader = branchLoaderMock

	branchLoaderMock.On("LoadBranches", config.Bitbucket).Return([]string{"master", "ets-123"}, nil)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		branchLoader = oldBranchLoader
		clock = oldClock
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
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/staging/databases/information_schema",
	"name": "ets-123",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/icuaZXnvrCc7-1kwIC7BwTFfgf0\"",
	"project": "test-project",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	},
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/staging/databases/foobar_database",
	"name": "foobar_database",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/TOxfaosOQfEN34mlC3_6aizAB4Q\"",
	"project": "test-project",
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
		Delete("/sql/v1beta4/projects/test-project/instances/testing/databases/foobar_database").
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

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Equal(t, "Operation delete of database failed\n", errOutput)
	assert.Contains(t, output, "Wait for operation delete of database to finish\n")
}

func TestCmdCleanup(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		Cluster: loader.Cluster{
			ProjectID: "test-project",
		},
		Database: loader.Database{
			Instance: "testing",
			BaseName: "base",
			PrefixBranchDatabase: "base_",
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
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/staging/databases/information_schema",
	"name": "base",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/icuaZXnvrCc7-1kwIC7BwTFfgf0\"",
	"project": "test-project",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	},
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/staging/databases/foobar_database",
	"name": "base_branch",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/TOxfaosOQfEN34mlC3_6aizAB4Q\"",
	"project": "test-project",
	"instance": "staging",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	},
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/staging/databases/foobar_database",
	"name": "foobar_database",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/TOxfaosOQfEN34mlC3_6aizAB4Q\"",
	"project": "test-project",
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
		Delete("/sql/v1beta4/projects/test-project/instances/testing/databases/base_branch").
		Reply(200).
		JSON(operationResponse)
	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCleanup, []string{"cleanup", "-c", "never.yml"})
	})

	assert.Empty(t, errOutput)

	assert.Equal(t, "Removed database base_branch", output)
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

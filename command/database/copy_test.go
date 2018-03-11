package database

import (
	"errors"
	"testing"

	"kube-helper/command"
	"kube-helper/loader"
	"kube-helper/mocks"

	"bufio"
	"compress/gzip"

	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"gopkg.in/h2non/gock.v1"
	util_clock "k8s.io/apimachinery/pkg/util/clock"
)

func TestCmdCopyWithWrongConfig(t *testing.T) {
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

	command.RunTestCommand(CmdCopy, []string{"copy", "-c", "never.yml", "foobar"})
}

func TestCmdCopyWithWrongSqlService(t *testing.T) {
	oldHandler := cli.OsExiter

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(loader.Config{}, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(mocks.ServiceBuilderInterface)
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

	command.RunTestCommand(CmdCopy, []string{"copy", "-c", "never.yml", "foobar"})
}

func TestCmdCommandWithExistingDatabase(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		Cluster: loader.Cluster{
			ProjectID: "test-project",
		},
		Database: loader.Database{
			Instance: "testing",
			BaseName: "foobar",
		},
	}

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(mocks.ServiceBuilderInterface)
	serviceBuilder = serviceBuilderMock

	sqlService, err := oldServiceBuilder.GetSqlService()
	serviceBuilderMock.On("GetSqlService").Return(sqlService, err)

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	defer gock.Off() // Flush pending mocks after test execution

	createAuthCall()

	response := `
	{
	"kind": "sql#database",
	"selfLink": "https://www.googleapis.com/sql/v1beta4/projects/testing/instances/testing/databases/foobar",
	"name": "foobar",
	"etag": "\"DlgRosmIegBpXj_rR5uyhdXAbP8/TOxfaosOQfEN34mlC3_6aizAB4Q\"",
	"project": "testing",
	"instance": "testing",
	"charset": "utf8",
	"collation": "utf8_general_ci"
	}]}
	`

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/instances/testing/databases/foobar").
		Reply(200).
		JSON(response)

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCopy, []string{"copy", "-c", "never.yml", "foobar-testing"})
	})

	assert.Equal(t, "Database foobar-testing already exists", errOutput)

	assert.Empty(t, output)

}

func TestCmdCommandWithFailureToGetStorageService(t *testing.T) {
	oldHandler := cli.OsExiter

	config := loader.Config{
		Cluster: loader.Cluster{
			ProjectID: "test-project",
		},
		Database: loader.Database{
			Instance: "testing",
			BaseName: "foobar",
			Bucket:   "foobar-testing",
		},
	}

	oldConfigLoader := configLoader
	configLoaderMock := new(mocks.ConfigLoaderInterface)

	configLoader = configLoaderMock

	configLoaderMock.On("LoadConfigFromPath", "never.yml").Return(config, nil)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(mocks.ServiceBuilderInterface)
	serviceBuilder = serviceBuilderMock

	appFS := afero.NewMemMapFs()
	oldFileSystem := fileSystem
	fileSystem = appFS

	file, err := appFS.OpenFile("copy.sql.gz", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)

	gzWriter := gzip.NewWriter(file)
	bufioWriter := bufio.NewWriter(gzWriter)

	bufioWriter.WriteString("CREATE DATABASE foobar;  \n\nUSE foobar; \n\n other stuff")

	bufioWriter.Flush()
	gzWriter.Close()
	file.Close()

	file, err = appFS.Open("copy.sql.gz")

	storageServiceMock := new(mocks.BucketServiceInterface)
	storageServiceMock.On("SetBucketACL", "bnuzuupn3haw4ioe66by@speckle-umbrella-3.iam.gserviceaccount.com", "WRITER").Return(nil)
	storageServiceMock.On("RemoveBucketACL", "bnuzuupn3haw4ioe66by@speckle-umbrella-3.iam.gserviceaccount.com").Return(nil)
	storageServiceMock.On("DownLoadFile", "foobar-testing.sql.gz", "bnuzuupn3haw4ioe66by@speckle-umbrella-3.iam.gserviceaccount.com").Return(bufio.NewReader(file), err)
	storageServiceMock.On("UploadFile", "foobar-testingtmp.sql.gz", "foobar-testingtmp.sql.gz", "bnuzuupn3haw4ioe66by@speckle-umbrella-3.iam.gserviceaccount.com").Return(nil)
	storageServiceMock.On("DeleteFile", "foobar-testingtmp.sql.gz").Return(nil)
	storageServiceMock.On("DeleteFile", "foobar-testing.sql.gz").Return(nil)

	sqlService, err := oldServiceBuilder.GetSqlService()
	serviceBuilderMock.On("GetSqlService").Return(sqlService, err)
	serviceBuilderMock.On("GetStorageService", "foobar-testing").Return(storageServiceMock, nil)

	oldClock := clock
	clock = util_clock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

	defer func() {
		cli.OsExiter = oldHandler
		configLoader = oldConfigLoader
		serviceBuilder = oldServiceBuilder
		fileSystem = oldFileSystem
		clock = oldClock
	}()

	cli.OsExiter = func(exitCode int) {
		assert.Equal(t, 1, exitCode)
	}

	createAuthCall()

	response := `
	{
	 "kind": "sql#instance",
	 "selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/testing",
	 "name": "testing",
	 "connectionName": "test-project:europe-west1:testing",
	 "etag": "\"7nzH-h2yIa30FGKFRs9YFu88s0g/MTAxMw\"",
	 "project": "test-project",
	 "state": "RUNNABLE",
	 "backendType": "SECOND_GEN",
	 "databaseVersion": "MYSQL_5_7",
	 "region": "europe-west1",
	 "settings": {
	  "kind": "sql#settings",
	  "settingsVersion": "1013",
	  "authorizedGaeApplications": [],
	  "tier": "db-g1-small",
	  "backupConfiguration": {
	   "kind": "sql#backupConfiguration",
	   "startTime": "23:00",
	   "enabled": false,
	   "binaryLogEnabled": false
	  },
	  "pricingPlan": "PER_USE",
	  "replicationType": "SYNCHRONOUS",
	  "activationPolicy": "ALWAYS",
	  "ipConfiguration": {
	   "ipv4Enabled": true,
	   "authorizedNetworks": []
	  },
	  "databaseFlags": [
	   {
		"name": "sql_mode",
		"value": "NO_AUTO_CREATE_USER"
	   }
	  ],
	  "dataDiskSizeGb": "10",
	  "dataDiskType": "PD_SSD",
	  "storageAutoResize": false,
	  "storageAutoResizeLimit": "0"
	 },
	 "serverCaCert": {
	  "kind": "sql#sslCert",
	  "instance": "testing",
	  "sha1Fingerprint": "4c3d951a0ef2767110471941cc70cb38c496da53",
	  "commonName": "C=US,O=Google\\, Inc,CN=Google Cloud SQL Server CA",
	  "certSerialNumber": "0",
	  "cert": "-----BEGIN CERTIFICATE-----\nMIIDITCCAgmgAwIBAgIBADANBgkqhQDExpHb29n\nbGUgQ2xvdWQgU1FMIFNlcnZlciBDQTEUMBIGA1UEChMLR29vZ2xlLCBJbmMxCzAJ\nBgNVBAYTAlVTMB4XDTE2MDkyMzA3NDkzOVoXDTE4MDkyMzA3NTAzOVowSDEjMCEG\nA1UEAxMaR29vZ2xlIENsb3VkIFNRTCBTZXJ2ZXIgQ0ExFDASBgNVBAoTC0dvb2ds\nZSwgSW5jMQswCQYDVQQGEwJVUzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC\nggEBAIXPL1fnFqNR0HwKJu8b8zTDti9ZnKogKZUuMwakC7w86IK0kBcOwxfc1msB\nHEnhyfaiQ+Jze+YQfqJB/VkQDjp6gneNEeKcm+kAVau8EaFMvTHX3sOktWYFLRLp\nc/fBTYqDGoxBw6Dw1RHVntV2xs3kYFdwNx3Gu6cb/wbHPXqxHJgbV/mQxmn8c+Ln\nJ5wdjDcV7BXK448OkaT5QTkaQc7Dx9b6zoh8UxLznyx8bQSVapbJbLxbFnWr1HsO\nvRy0M1xq1rZ4AT6Qi4bm3CnXf+kg2P9qgRFKr71xZRzEtAJcA+N/6y7xHYAoNYu0\n3XMTme+rCla54Q/tyo9/bnqOv0cCAwEAAaMWMBQwEgYDVR0TAQH/BAgwBgEB/wIB\nADANBgkqhkiG9w0BAQUFAAOCAQEAdBuxPVnjqZ16fZs9TVFALFWBbzVllKizoR0E\nztd0cvGLEx33ewkxwVtcd+I48uODWtMu7PltW3Re2/CI0J733bD2lEiO0rQumszV\nev6TwRL/AoRzQIRVHBgkHJ31UBNlwP+m/r6QvyX8jJhF4wFkPj9N5wgF7F2DPFdR\ni2uR1DVTUMwRrGG1GL1p1UP2k19fYlafomMVJbAuZD2gq1vELZPK3bHwoXDqruyZ\naME4ZLw4ULOcCUkCF0ABN9himz7mQyovj4MYNpRocySGus3ZR6eEAXcfJQ8EP4F2\nxRsb0/RRP9dE/U0OYRG8r9u3AypHH3vkTuoO8fE3Te01YtLe9Q==\n-----END CERTIFICATE-----",
	  "createTime": "2016-09-23T07:49:39.699Z",
	  "expirationTime": "2018-09-23T07:50:39.699Z"
	 },
	 "ipAddresses": [
	  {
	   "ipAddress": "23.251.140.185",
	   "type": "PRIMARY"
	  }
	 ],
	 "instanceType": "CLOUD_SQL_INSTANCE",
	 "serviceAccountEmailAddress": "bnuzuupn3haw4ioe66by@speckle-umbrella-3.iam.gserviceaccount.com"
	}
	`

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/instances/testing/databases/foobar").
		Reply(404)

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/instances/testing").
		Reply(200).
		JSON(response)

	exportRespone := `
	{
	 "kind": "sql#operation",
	 "selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/operations/0b213b5f-d400-451f-8355-8b9f7d80ba90",
	 "targetProject": "test-project",
	 "targetId": "testing",
	 "targetLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/testing",
	 "name": "0b213b5f-d400-451f-8355-8b9f7d80ba90",
	 "operationType": "EXPORT",
	 "status": "PENDING",
	 "user": "alexander.miehe@tourstream.eu",
	 "insertTime": "2017-07-10T07:57:40.639Z",
	 "exportContext": {
	  "kind": "sql#exportContext",
	  "uri": "gs://foobar-testing/dummy.sql.gz",
	  "databases": [
	   "testing"
	  ],
	  "sqlExportOptions": {
	   "schemaOnly": false
	  }
	 }
	}
	`

	gock.New("https://www.googleapis.com").
		Post("/sql/v1beta4/projects/test-project/instances/testing/export").
		Reply(200).
		JSON(exportRespone)

	operationStatus := `
	{
	 "kind": "sql#operation",
	 "selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/operations/0b213b5f-d400-451f-8355-8b9f7d80ba90",
	 "targetProject": "test-project",
	 "targetId": "testing",
	 "targetLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/testing",
	 "name": "0b213b5f-d400-451f-8355-8b9f7d80ba90",
	 "operationType": "EXPORT",
	 "status": "DONE",
	 "user": "alexander.miehe@tourstream.eu",
	 "insertTime": "2017-07-10T07:57:40.639Z",
	 "startTime": "2017-07-10T07:57:40.805Z",
	 "endTime": "2017-07-10T07:57:40.805Z",
	 "exportContext": {
	  "kind": "sql#exportContext",
	  "uri": "gs://foobar-testing/dummy.sql.gz",
	  "databases": [
	   "testing"
	  ],
	  "sqlExportOptions": {
	   "schemaOnly": false
	  }
	 }
	}
	`

	gock.New("https://www.googleapis.com").
		Get("/sql/v1beta4/projects/test-project/operations/0b213b5f-d400-451f-8355-8b9f7d80ba90").
		Persist().
		Reply(200).
		JSON(operationStatus)

	importRespone := `
	{
	 "kind": "sql#operation",
	 "selfLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/operations/0b213b5f-d400-451f-8355-8b9f7d80ba90",
	 "targetProject": "test-project",
	 "targetId": "testing",
	 "targetLink": "https://www.googleapis.com/sql/v1beta4/projects/test-project/instances/testing",
	 "name": "0b213b5f-d400-451f-8355-8b9f7d80ba90",
	 "operationType": "IMPORT",
	 "status": "PENDING",
	 "user": "alexander.miehe@tourstream.eu",
	 "insertTime": "2017-07-10T07:57:40.639Z",
	 "importContext": {
	  "kind": "sql#importContext",
	  "uri": "gs://foobar-testing/dummy.sql.gz",
	  "database": "other"
	 }
	}`

	gock.New("https://www.googleapis.com").
		Post("/sql/v1beta4/projects/test-project/instances/testing/databases").
		Reply(201).
		JSON(importRespone)

	gock.New("https://www.googleapis.com").
		Post("/sql/v1beta4/projects/test-project/instances/testing/import").
		Reply(201).
		JSON(importRespone)

	output, errOutput := captureOutput(func() {
		command.RunTestCommand(CmdCopy, []string{"copy", "-c", "never.yml", "foobar-testing"})
	})

	assert.Empty(t, errOutput)

	tmpFile, err := appFS.Open("foobar-testingtmp.sql.gz")

	assert.NoError(t, err)

	gz, err := gzip.NewReader(tmpFile)

	assert.NoError(t, err)

	defer gz.Close()

	scanner := bufio.NewScanner(gz)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	assert.Equal(t, lines, []string{"CREATE DATABASE foobar-testing;  ", "", "USE foobar-testing; ", "", " other stuff"})

	assert.Equal(t, output, "Wait for operation export of database to finish\nExport for sql finished\nWait for operation creation of database to finish\nWait for operation import of database to finish\nImport for sql finished\n")
}

func createAuthCall() {
	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Persist().
		Reply(200).
		JSON(map[string]string{"access_token": "bar"})
}

/**
"error": {
  "kind": "sql#operationErrors",
  "errors": [
   {
    "kind": "sql#operationError",
    "code": "UNKNOWN"
   }
  ]
 },
*/

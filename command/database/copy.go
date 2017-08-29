package database

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"kube-helper/loader"
	"kube-helper/util"
	"strings"

	"github.com/urfave/cli"
	"google.golang.org/api/sqladmin/v1beta4"
	"github.com/spf13/afero"
)

const filenamePattern = "%s.sql.gz"
const filePathPattern = "gs://%s/%s"

var fileSystem = afero.NewOsFs()

func CmdCopy(c *cli.Context) error {
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return CopyDatabaseByBranchName(c.Args().Get(0), configContainer)
}

func GetDatabaseName(databaseConfig loader.Database, branchName string) string {
	if branchName == "master" {
		return databaseConfig.BaseName
	}
	databaseName := databaseConfig.PrefixBranchDatabase + branchName
	length := 60
	if length > len(databaseName) {
		length = len(databaseName)
	}
	return databaseName[:length]
}

func CopyDatabaseByBranchName(branchName string, configContainer loader.Config) error {

	databaseName := GetDatabaseName(configContainer.Database, branchName)

	if databaseName == configContainer.Database.BaseName {
		fmt.Fprint(cli.ErrWriter,"Copy to the same database makes no sense")
		return nil
	}

	sqlService, err := serviceBuilder.GetSqlService()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	database, _ := sqlService.Databases.Get(configContainer.ProjectID, configContainer.Database.Instance, databaseName).Do()
	if database != nil {
		fmt.Fprintf(cli.ErrWriter,"Database %s already exists", databaseName)
		return nil
	}
	storageService, err := serviceBuilder.GetStorageService(configContainer.Database.Bucket)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	instance, err := sqlService.Instances.Get(configContainer.ProjectID, configContainer.Database.Instance).Do()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = storageService.SetBucketACL(instance.ServiceAccountEmailAddress, "WRITER")

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	dumpFilename := fmt.Sprintf(filenamePattern, databaseName)
	exportFilePath := fmt.Sprintf(filePathPattern, configContainer.Database.Bucket, dumpFilename)

	exportRequest := &sqladmin.InstancesExportRequest{}
	exportRequest.ExportContext = &sqladmin.ExportContext{}
	exportRequest.ExportContext.Databases = append(exportRequest.ExportContext.Databases, configContainer.Database.BaseName)
	exportRequest.ExportContext.Uri = exportFilePath

	operation, err := sqlService.Instances.Export(configContainer.ProjectID, configContainer.Database.Instance, exportRequest).Do()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "export of database")

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	defer storageService.DeleteFile(dumpFilename)

	fmt.Fprintln(writer,"Export for sql finished")

	downloadedFile, err := storageService.DownLoadFile(dumpFilename, instance.ServiceAccountEmailAddress)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	gz, err := gzip.NewReader(downloadedFile)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	defer gz.Close()

	scanner := bufio.NewScanner(gz)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	tmpName := fmt.Sprintf(filenamePattern, databaseName+"tmp")
	gzWriter, err := util.CreateGzWriter(fileSystem, tmpName)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	scanner.Err()

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "CREATE DATABASE") || strings.HasPrefix(line, "USE") {
			line = strings.Replace(line, configContainer.Database.BaseName, databaseName, 1)
		}
		_, err = gzWriter.Write(line + "\n")
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	gzWriter.Close()
	err = storageService.UploadFile(tmpName, tmpName, instance.ServiceAccountEmailAddress)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	defer storageService.DeleteFile(tmpName)

	operation, err = sqlService.Databases.Insert(configContainer.ProjectID, configContainer.Database.Instance, &sqladmin.Database{
		Name: databaseName,
	}).Do()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "creation of database")

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	importFilePath := fmt.Sprintf(filePathPattern, configContainer.Database.Bucket, tmpName)

	importRequest := &sqladmin.InstancesImportRequest{}
	importRequest.ImportContext = &sqladmin.ImportContext{
		Database: databaseName,
		FileType: "SQL",
		Uri:      importFilePath,
	}
	operation, err = sqlService.Instances.Import(configContainer.ProjectID, configContainer.Database.Instance, importRequest).Do()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "import of database")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	fmt.Fprintln(writer, "Import for sql finished")

	return storageService.RemoveBucketACL(instance.ServiceAccountEmailAddress)
}

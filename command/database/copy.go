package database

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"strings"
	"github.com/urfave/cli"
	"google.golang.org/api/sqladmin/v1beta4"
	"kube-helper/loader"
	"kube-helper/util"
)

const filenamePattern = "%s.sql.gz"
const filepathPattern = "gs://%s/%s"

func CmdCopy(c *cli.Context) error {
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
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

	sqlService, err := serviceBuilder.GetSqlService()

	if err != nil {
		return err
	}

	database, _ := sqlService.Databases.Get(configContainer.ProjectID, configContainer.Database.Instance, databaseName).Do()
	if database != nil {
		log.Printf("Database %s already exists", databaseName)
		return nil
	}
	storageService, err := serviceBuilder.GetStorageService(configContainer.Database.Bucket)

	if err != nil {
		return err
	}

	instance, err := sqlService.Instances.Get(configContainer.ProjectID, configContainer.Database.Instance).Do()

	if err != nil {
		return err
	}

	err = storageService.SetBucketACL(instance.ServiceAccountEmailAddress, "WRITER")

	if err != nil {
		return err
	}

	dumpFilename := fmt.Sprintf(filenamePattern, databaseName)
	exportFilePath := fmt.Sprintf(filepathPattern, configContainer.Database.Bucket, dumpFilename)

	exportRequest := &sqladmin.InstancesExportRequest{}
	exportRequest.ExportContext = &sqladmin.ExportContext{}
	exportRequest.ExportContext.Databases = append(exportRequest.ExportContext.Databases, configContainer.Database.BaseName)
	exportRequest.ExportContext.Uri = exportFilePath

	operation, err := sqlService.Instances.Export(configContainer.ProjectID, configContainer.Database.Instance, exportRequest).Do()

	if err != nil {
		return err
	}

	err = waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "export of database")

	if err != nil {
		return err
	}

	log.Print("Export for sql finished")

	tmpDownloadedFile, err := storageService.DownLoadFile(dumpFilename, instance.ServiceAccountEmailAddress)

	if err != nil {
		return err
	}

	gz, err := gzip.NewReader(tmpDownloadedFile)

	if err != nil {
		return err
	}

	defer gz.Close()

	scanner := bufio.NewScanner(gz)
	tmpName := fmt.Sprintf(filenamePattern, databaseName+"tmp")
	writer, err := util.CreateGzWriter(tmpName)

	if err != nil {
		return err
	}

	defer writer.Close()

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "CREATE DATABASE") || strings.HasPrefix(line, "USE") {
			line = strings.Replace(line, configContainer.Database.BaseName, databaseName, 1)
		}

		writer.Write(line+"\n")
	}

	err = storageService.UploadFile(tmpName, tmpName, instance.ServiceAccountEmailAddress)

	if err != nil {
		return err
	}

	operation, err = sqlService.Databases.Insert(configContainer.ProjectID, configContainer.Database.Instance, &sqladmin.Database{
		Name: databaseName,
	}).Do()

	if err != nil {
		return err
	}

	err = waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "creation of database")

	if err != nil {
		return err
	}

	importFilePath := fmt.Sprintf(filepathPattern, configContainer.Database.Bucket, tmpName)

	importRequest := &sqladmin.InstancesImportRequest{}
	importRequest.ImportContext = &sqladmin.ImportContext{
		Database: databaseName,
		FileType: "SQL",
		Uri:      importFilePath,
	}
	operation, err = sqlService.Instances.Import(configContainer.ProjectID, configContainer.Database.Instance, importRequest).Do()

	if err != nil {
		return err
	}

	err = waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "import of database")
	if err != nil {
		return err
	}
	log.Print("Import for sql finished")
	err = storageService.RemoveBucketACL(instance.ServiceAccountEmailAddress)
	if err != nil {
		return err
	}

	err = storageService.DeleteFile(dumpFilename)
	if err != nil {
		return err
	}
	return storageService.DeleteFile(tmpName)
}

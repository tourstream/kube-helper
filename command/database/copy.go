package database

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	GoStorage "cloud.google.com/go/storage"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
	"google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
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
	storageService, err := serviceBuilder.GetStorageService()

	if err != nil {
		return err
	}

	instance, err := sqlService.Instances.Get(configContainer.ProjectID, configContainer.Database.Instance).Do()

	if err != nil {
		return err
	}

	err = setBucketACL(storageService, configContainer.Database.Bucket, instance.ServiceAccountEmailAddress, "WRITER")

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

	err = setBucketACL(storageService, configContainer.Database.Bucket, instance.ServiceAccountEmailAddress, "READER")

	if err != nil {
		return err
	}

	//download file

	bucket, err := storageService.Objects.Get(configContainer.Database.Bucket, dumpFilename).Do()

	if err != nil {
		return err
	}

	err = downloadFromUrl(bucket.MediaLink, dumpFilename)

	if err != nil {
		return err
	}

	file, err := os.Open(dumpFilename)

	if err != nil {
		return err
	}

	gz, err := gzip.NewReader(file)

	if err != nil {
		return err
	}

	defer file.Close()
	defer gz.Close()

	scanner := bufio.NewScanner(gz)
	tmpName := fmt.Sprintf(filenamePattern, databaseName+"tmp")
	f := util.CreateGZ(tmpName)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "CREATE DATABASE") || strings.HasPrefix(line, "USE") {
			line = strings.Replace(line, configContainer.Database.BaseName, databaseName, 1)
		}

		util.WriteGZ(f, line+"\n")

	}

	util.CloseGZ(f)

	storageHelper, err := GoStorage.NewClient(context.Background())

	if err != nil {
		return err
	}

	w := storageHelper.Bucket(configContainer.Database.Bucket).Object(tmpName).NewWriter(context.Background())
	w.ACL = []GoStorage.ACLRule{{Entity: GoStorage.ACLEntity("user-" + instance.ServiceAccountEmailAddress), Role: GoStorage.RoleReader}}

	file2, err := os.Open(tmpName)

	if err != nil {
		return err
	}

	_, err = io.Copy(w, file2)

	if err != nil {
		return err
	}

	err = w.Close()

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
	err = removeBucketACL(storageService, configContainer.Database.Bucket, instance.ServiceAccountEmailAddress)
	if err != nil {
		return err
	}

	err = storageService.Objects.Delete(configContainer.Database.Bucket, dumpFilename).Do()
	if err != nil {
		return err
	}
	return storageService.Objects.Delete(configContainer.Database.Bucket, tmpName).Do()
}

func downloadFromUrl(url string, filename string) error {
	fmt.Println("Downloading", url, "to", filename)

	client, err := serviceBuilder.GetClient(sqladmin.CloudPlatformScope)
	if err != nil {
		return err
	}

	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()

	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	_, err = io.Copy(output, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func setBucketACL(storageService *storage.Service, bucket string, serviceAccount string, role string) error {
	_, err := storageService.BucketAccessControls.Insert(bucket, &storage.BucketAccessControl{
		Email:  serviceAccount,
		Entity: "user-" + serviceAccount,
		Role:   role,
	}).Do()

	if err != nil {
		return err
	}

	return nil
}

func removeBucketACL(storageService *storage.Service, bucket string, serviceAccount string) error {
	return storageService.BucketAccessControls.Delete(bucket, "user-"+serviceAccount).Do()
}
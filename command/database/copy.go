package database

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	GoStorage "cloud.google.com/go/storage"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
	"kube-helper/util"
)

const filenamePattern = "%s.sql.gz"
const filepathPattern = "gs://%s/%s"

func CmdCopy(c *cli.Context) error {
	configContainer, _ := util.LoadConfigFromPath(c.String("config"))

	return CopyDatabaseByBranchName(c.Args().Get(0), configContainer)
}

func GetDatabaseName(databaseConfig util.Database, branchName string) string {
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

func CopyDatabaseByBranchName(branchName string, configContainer util.Config) error {

	databaseName := GetDatabaseName(configContainer.Database, branchName)

	sqlService := createSqlService()

	database, _ := sqlService.Databases.Get(configContainer.ProjectID, configContainer.Database.Instance, databaseName).Do()
	if database != nil {
		log.Printf("Database %s already exists", databaseName)
		return nil
	}
	storageService := createStorageService()
	instance, err := sqlService.Instances.Get(configContainer.ProjectID, configContainer.Database.Instance).Do()

	util.CheckError(err)

	setBucketACL(storageService, configContainer.Database.Bucket, instance.ServiceAccountEmailAddress, "WRITER")
	dumpFilename := fmt.Sprintf(filenamePattern, databaseName)
	exportFilePath := fmt.Sprintf(filepathPattern, configContainer.Database.Bucket, dumpFilename)

	exportRequest := &sqladmin.InstancesExportRequest{}
	exportRequest.ExportContext = &sqladmin.ExportContext{}
	exportRequest.ExportContext.Databases = append(exportRequest.ExportContext.Databases, configContainer.Database.BaseName)
	exportRequest.ExportContext.Uri = exportFilePath

	operation, err := sqlService.Instances.Export(configContainer.ProjectID, configContainer.Database.Instance, exportRequest).Do()
	checkError(err)
	waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "export of database")
	log.Print("Export for sql finished")

	setBucketACL(storageService, configContainer.Database.Bucket, instance.ServiceAccountEmailAddress, "READER")

	//download file

	bucket, err := storageService.Objects.Get(configContainer.Database.Bucket, dumpFilename).Do()
	util.CheckError(err)

	downloadFromUrl(bucket.MediaLink, dumpFilename)

	file, err := os.Open(dumpFilename)
	util.CheckError(err)

	gz, err := gzip.NewReader(file)

	util.CheckError(err)

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
	util.CheckError(err)

	w := storageHelper.Bucket(configContainer.Database.Bucket).Object(tmpName).NewWriter(context.Background())
	w.ACL = []GoStorage.ACLRule{{Entity: GoStorage.ACLEntity("user-" + instance.ServiceAccountEmailAddress), Role: GoStorage.RoleReader}}

	file2, err := os.Open(tmpName)
	util.CheckError(err)

	io.Copy(w, file2)

	w.Close()

	operation, err = sqlService.Databases.Insert(configContainer.ProjectID, configContainer.Database.Instance, &sqladmin.Database{
		Name: databaseName,
	}).Do()

	checkError(err)
	waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "creation of database")

	importFilePath := fmt.Sprintf(filepathPattern, configContainer.Database.Bucket, tmpName)

	importRequest := &sqladmin.InstancesImportRequest{}
	importRequest.ImportContext = &sqladmin.ImportContext{
		Database: databaseName,
		FileType: "SQL",
		Uri:      importFilePath,
	}
	operation, err = sqlService.Instances.Import(configContainer.ProjectID, configContainer.Database.Instance, importRequest).Do()
	checkError(err)
	waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "import of database")
	log.Print("Import for sql finished")
	removeBucketACL(storageService, configContainer.Database.Bucket, instance.ServiceAccountEmailAddress)

	err = storageService.Objects.Delete(configContainer.Database.Bucket, dumpFilename).Do()
	err = storageService.Objects.Delete(configContainer.Database.Bucket, tmpName).Do()
	checkError(err)
	return nil
}

func downloadFromUrl(url string, filename string) {
	fmt.Println("Downloading", url, "to", filename)

	client := getClient()

	output, err := os.Create(filename)
	if err != nil {
		log.Fatal("Error while creating", filename, "-", err)

	}
	defer output.Close()

	response, err := client.Get(url)
	if err != nil {
		log.Fatal("Error while downloading", url, "-", err)
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		log.Fatal("Error while downloading", url, "-", err)
	}

	fmt.Println(n, "bytes downloaded.")
}

func waitForOperationToFinish(sqlService *sqladmin.Service, operation *sqladmin.Operation, projectID string, operationType string) {
	var err error
	for {
		if operation.Status == "DONE" {
			if operation.Error != nil && len(operation.Error.Errors) > 0 {
				for _, err := range operation.Error.Errors {
					log.Print(err)
				}
				log.Panicf("Operation %s failed", operationType)
			}
			break
		}
		operation, err = sqlService.Operations.Get(projectID, operation.Name).Do()
		checkError(err)
		log.Printf("Wait for operation %s to finish", operationType)
		time.Sleep(time.Second * 5)
	}
}

func setBucketACL(storageService *storage.Service, bucket string, serviceAccount string, role string) {
	_, err := storageService.BucketAccessControls.Insert(bucket, &storage.BucketAccessControl{
		Email:  serviceAccount,
		Entity: "user-" + serviceAccount,
		Role:   role,
	}).Do()

	checkError(err)
}

func removeBucketACL(storageService *storage.Service, bucket string, serviceAccount string) {
	err := storageService.BucketAccessControls.Delete(bucket, "user-"+serviceAccount).Do()
	checkError(err)
}

func checkError(e error) {
	if e != nil {
		log.Panic(e)
	}
}

func getClient() *http.Client {
	ctx := context.Background()

	client, err := google.DefaultClient(ctx, storage.CloudPlatformScope)
	util.CheckError(err)

	return client
}

func createStorageService() *storage.Service {
	storageService, err := storage.New(getClient())
	checkError(err)

	return storageService
}

func createSqlService() *sqladmin.Service {
	ctx := context.Background()

	client, err := google.DefaultClient(ctx, sqladmin.CloudPlatformScope)
	checkError(err)
	sqlService, err := sqladmin.New(client)
	checkError(err)

	return sqlService
}

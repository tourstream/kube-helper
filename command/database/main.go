package database

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"google.golang.org/api/sqladmin/v1beta4"
	"kube-helper/loader"
	"kube-helper/service"
)

var writer io.Writer = os.Stdout
var serviceBuilder service.BuilderInterface = new(service.Builder)
var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var branchLoader loader.BranchLoaderInterface = new(loader.BranchLoader)

func waitForOperationToFinish(sqlService *sqladmin.Service, operation *sqladmin.Operation, projectID string, operationType string) error {
	var err error
	for {
		if operation.Status == "DONE" {
			if operation.Error != nil && len(operation.Error.Errors) > 0 {
				for _, err := range operation.Error.Errors {
					fmt.Fprint(writer, err)
				}
				return errors.New(fmt.Sprintf("Operation %s failed", operationType))
			}
			break
		}
		operation, err = sqlService.Operations.Get(projectID, operation.Name).Do()

		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Wait for operation %s to finish", operationType)
		time.Sleep(time.Second * 5)
	}
	return nil
}

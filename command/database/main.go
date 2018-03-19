package database

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"kube-helper/loader"
	"kube-helper/service/builder"

	"google.golang.org/api/sqladmin/v1beta4"
	utilClock "k8s.io/apimachinery/pkg/util/clock"
)

var writer io.Writer = os.Stdout
var serviceBuilder = builder.NewServiceBuilder()
var configLoader = loader.NewConfigLoader()
var branchLoader loader.BranchLoaderInterface = new(loader.BranchLoader)
var clock utilClock.Clock = new(utilClock.RealClock)

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

		fmt.Fprintln(writer, fmt.Sprintf("Wait for operation %s to finish", operationType))
		clock.Sleep(time.Second * 5)
	}
	return nil
}

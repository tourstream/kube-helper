package database

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
	"google.golang.org/api/sqladmin/v1beta4"
	"kube-helper/util"
)

var cleanUpExcludes = []string{"information_schema", "mysql", "performance_schema", "sys"}

func CmdCleanup(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	sqlService, err := serviceBuilder.GetSqlService()

	if err != nil {
		return err
	}

	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return err
	}
	databases, err := getDatabases(sqlService, configContainer.ProjectID, configContainer.Database.Instance)

	if err != nil {
		return err
	}

	for _, database := range databases {
		if database == configContainer.Database.BaseName {
			continue
		}

		branch := strings.TrimPrefix(database, configContainer.Database.PrefixBranchDatabase)

		if util.InArray(branches, branch) == false {
			operation, err := sqlService.Databases.Delete(configContainer.ProjectID, configContainer.Database.Instance, database).Do()
			if err != nil {
				return err
			}
			err = waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "delete of database")
			if err != nil {
				return err
			}

			fmt.Fprintf(writer, "Removed database %s", database)
		}
	}

	return nil
}

func getDatabases(sqlService *sqladmin.Service, projectID string, instance string) ([]string, error) {
	list, err := sqlService.Databases.List(projectID, instance).Do()
	databases := []string{}

	if err != nil {
		return nil, err
	}

	for _, database := range list.Items {
		if util.InArray(cleanUpExcludes, database.Name) == false {
			databases = append(databases, database.Name)
		}
	}

	return databases, nil
}

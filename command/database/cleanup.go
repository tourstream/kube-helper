package database

import (
	"log"
	"strings"

	"github.com/urfave/cli"

	"google.golang.org/api/sqladmin/v1beta4"

	"kube-helper/config"
	"kube-helper/util"
)

var cleanUpExcludes = []string{"information_schema", "mysql", "performance_schema", "sys"}

func CmdCleanup(c *cli.Context) error {

	configContainer := config.LoadConfigFromPath(c.String("config"))
	sqlService := createSqlService()
	branches := util.GetBranches(configContainer.Cleanup.RepoUrl)
	databases := getDatabases(sqlService, configContainer.ProjectID, configContainer.Database.Instance)

	for _, database := range databases {
		if (database == configContainer.Database.BaseName) {
			continue
		}

		branch := strings.TrimPrefix(database, configContainer.Database.PrefixBranchDatabase)

		if (util.InArray(branches, branch) == false) {
			operation, err := sqlService.Databases.Delete(configContainer.ProjectID, configContainer.Database.Instance, database).Do()
			checkError(err)
			waitForOperationToFinish(sqlService, operation, configContainer.ProjectID, "delete of database")
			log.Printf("Removed database %s", database)
		}
	}

	return nil
}

func getDatabases(sqlService *sqladmin.Service, projectID string, instance string) []string {
	list, err := sqlService.Databases.List(projectID, instance).Do()
	databases := []string{}
	util.CheckError(err)
	for _, database := range list.Items {
		if (util.InArray(cleanUpExcludes, database.Name) == false) {
			databases = append(databases, database.Name)
		}
	}

	return databases
}

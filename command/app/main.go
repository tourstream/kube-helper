package app

import (
	"io"
	"os"

	"kube-helper/loader"
	"kube-helper/service/app"
	"kube-helper/service/builder"
	"strings"
)

var writer io.Writer = os.Stdout
var serviceBuilder builder.ServiceBuilderInterface = new(builder.Builder)
var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var branchLoader loader.BranchLoaderInterface = new(loader.BranchLoader)
var applicationServiceCreator = app.NewApplicationService

func getNamespace(branchName string, isProdution bool) string {
	namespace := strings.ToLower(branchName)

	if isProdution {
		return loader.ProductionEnvironment
	}

	if namespace == "" || namespace == loader.StagingEnvironment || namespace == "master" {
		namespace = loader.StagingEnvironment
	}

	return namespace
}

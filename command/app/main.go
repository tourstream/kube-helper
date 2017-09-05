package app

import (
	"io"
	"os"

	"kube-helper/loader"
	"kube-helper/service"
	"strings"
)

var writer io.Writer = os.Stdout
var serviceBuilder service.BuilderInterface = new(service.Builder)
var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var branchLoader loader.BranchLoaderInterface = new(loader.BranchLoader)

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

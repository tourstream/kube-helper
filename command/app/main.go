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
var imagesService service.ImagesInterface = new(service.Images)

const stagingEnvironment = "staging"

func getNamespace(branchName string) string {
	namespace := strings.ToLower(branchName)

	if namespace == "" || namespace == stagingEnvironment || namespace == "master" {
		namespace = stagingEnvironment
	}

	return namespace
}

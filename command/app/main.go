package app

import (
	"io"
	"os"

	"kube-helper/loader"
	"kube-helper/service"
)

var writer io.Writer = os.Stdout
var serviceBuilder service.BuilderInterface = new(service.Builder)
var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var branchLoader loader.BranchLoaderInterface = new(loader.BranchLoader)
var imagesService service.ImagesInterface = new(service.Images)

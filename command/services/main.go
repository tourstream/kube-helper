package services

import (
	"io"
	"kube-helper/loader"
	"kube-helper/service/builder"
	"os"
)

var writer io.Writer = os.Stdout
var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var serviceBuilder builder.ServiceBuilderInterface = new(builder.Builder)

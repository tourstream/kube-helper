package services

import (
	"io"
	"kube-helper/loader"
	"kube-helper/service/builder"
	"os"
)

var writer io.Writer = os.Stdout
var configLoader = loader.NewConfigLoader()
var serviceBuilder = builder.NewServiceBuilder()

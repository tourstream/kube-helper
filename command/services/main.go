package services

import (
	"kube-helper/loader"
	"kube-helper/service"
	"os"
	"io"
	"strings"
)

var writer io.Writer = os.Stdout
var configLoader loader.ConfigLoaderInterface = new(loader.Config)
var serviceBuilder service.BuilderInterface = new(service.Builder)
var projectId = ""

func getResourceName(resourceLink string) string {
	parts := strings.Split(resourceLink, "/")
	return parts[len(parts)-1]
}
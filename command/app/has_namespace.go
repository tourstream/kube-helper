package app

import (
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli"
	"k8s.io/client-go/kubernetes"
	"kube-helper/loader"
	"kube-helper/service"
)

var Writer io.Writer = os.Stdout

var serviceBuilder service.BuilderInterface = new(service.Builder)
var configLoader loader.ConfigLoaderInterface = new(loader.Config)

func CmdHasNamespace(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, _ := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	if hasNameSpace(clientSet, getNamespace(c.Args().Get(0))) == false {
		fmt.Fprint(Writer, "false")
		return nil
	}

	fmt.Fprint(Writer, "true")

	return nil
}

func hasNameSpace(clientSet kubernetes.Interface, namespace string) bool {
	_, err := clientSet.CoreV1().Namespaces().Get(namespace)

	if err != nil {
		return false
	}

	return true

}

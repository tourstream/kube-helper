package app

import (
	"fmt"

	"github.com/urfave/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CmdShutdownAll(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	if err != nil {
		return err
	}

	list, err := clientSet.CoreV1().Namespaces().List(v1.ListOptions{})

	if err != nil {
		return err
	}

	for _, namespace := range list.Items {
		if namespace.Name == "kube-system" || namespace.Name == "default" {
			continue
		}

		appService, err := serviceBuilder.GetApplicationService(clientSet, namespace.Name, configContainer)

		if err != nil {
			return err
		}

		err = appService.DeleteByNamespace()

		if err != nil {
			fmt.Fprint(writer, err)
		}
	}

	return nil
}

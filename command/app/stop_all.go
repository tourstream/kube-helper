package app

import (
	"log"

	"github.com/urfave/cli"
	"k8s.io/client-go/pkg/api/v1"
	"kube-helper/util"
)

func CmdShutdownAll(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, _ := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	list, err := clientSet.CoreV1().Namespaces().List(v1.ListOptions{})
	util.CheckError(err)

	for _, namespace := range list.Items {
		if namespace.Name == "kube-system" || namespace.Name == "default" {
			continue
		}
		err := deleteApplicationByNamespace(clientSet, namespace.Name, configContainer)

		if err != nil {
			log.Print(err)
		}
	}

	return nil
}

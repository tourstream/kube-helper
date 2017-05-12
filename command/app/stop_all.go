package app

import (
	"github.com/urfave/cli"

	"k8s.io/client-go/pkg/api/v1"

	"kube-helper/util"
)

func CmdShutdownAll(c *cli.Context) error {

	configContainer, err := util.LoadConfigFromPath(c.String("config"))
	if err != nil {
		return err
	}
	createContainerService()
	createClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	list, err := clientset.CoreV1().Namespaces().List(v1.ListOptions{})
	util.CheckError(err)

	for _, namespace := range list.Items {
		if namespace.Name == "kube-system" || namespace.Name == "default" {
			continue
		}
		deleteApplicationByNamespace(namespace.Name, configContainer)
	}

	return nil
}

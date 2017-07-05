package app

import (
	"github.com/urfave/cli"
)

func CmdShutdown(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0))
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	if err != nil {
		return err
	}

	appService, err := serviceBuilder.GetApplicationService(clientSet, kubernetesNamespace, configContainer)

	if err != nil {
		return err
	}

	return appService.DeleteByNamespace()
}

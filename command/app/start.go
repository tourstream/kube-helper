package app

import (
	"github.com/urfave/cli"
)

func CmdStartUp(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0), c.Bool("production"))

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

	return appService.CreateForNamespace()
}

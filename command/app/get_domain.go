package app

import (
	"fmt"
	"github.com/urfave/cli"
)

func CmdGetDomain(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	if err != nil {
		return err
	}

	appService, err := serviceBuilder.GetApplicationService(clientSet, getNamespace(c.Args().Get(0), false), configContainer)

	if err != nil {
		return err
	}

	fmt.Fprint(writer, appService.GetDomain(configContainer.DNS))
	return nil
}

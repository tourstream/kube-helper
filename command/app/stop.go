package app

import (
	"github.com/urfave/cli"
)

func CmdShutdown(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0), c.Bool("production"), c.String("namespace"))
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	configContainer.Internal.IsProduction = c.Bool("production")

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	appService, err := serviceBuilder.GetApplicationService(clientSet, kubernetesNamespace, configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = appService.DeleteByNamespace()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

package app

import (
	"github.com/urfave/cli"
)

func CmdShutdown(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0), c.Bool("production"))
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	appService, err := applicationServiceCreator(kubernetesNamespace, configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = appService.DeleteByNamespace()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

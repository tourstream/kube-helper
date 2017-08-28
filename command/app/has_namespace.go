package app

import (
	"fmt"
	"github.com/urfave/cli"
)

func CmdHasNamespace(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	appService, err := serviceBuilder.GetApplicationService(clientSet, getNamespace(c.Args().Get(0), false), configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if appService.HasNamespace() == false {
		fmt.Fprint(writer, "false")
		return nil
	}

	fmt.Fprint(writer, "true")

	return nil
}

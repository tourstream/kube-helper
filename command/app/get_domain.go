package app

import (
	"fmt"

	"github.com/urfave/cli"
)

func CmdGetDomain(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	appService, err := applicationServiceCreator(getNamespace(c.Args().Get(0), false, ""), configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	fmt.Fprint(writer, appService.GetDomain(configContainer.DNS))
	return nil
}

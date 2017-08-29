package app

import (
	"github.com/urfave/cli"
	"fmt"
	"kube-helper/loader"
)

func CmdApply(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0), c.Bool("production"))

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

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

	tag := "staging-" + kubernetesNamespace + "-latest"
	if kubernetesNamespace == loader.StagingEnvironment {
		tag = "staging-latest"
	}

	if kubernetesNamespace == loader.ProductionEnvironment {
		tag = "latest"
	}

	hasTag, err := imagesService.HasTag(configContainer.Cleanup, tag)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if !hasTag {
		return cli.NewExitError(fmt.Sprintf("No Image '%s' found for namespace '%s' ",tag, kubernetesNamespace), 0)
	}


	err = appService.Apply()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

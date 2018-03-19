package app

import (
	"fmt"
	"kube-helper/loader"

	"github.com/urfave/cli"
)

// CmdApply applies a configuration to kubernetes, this works as an upsert.
func CmdApply(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0), c.Bool("production"), c.String("namespace"))

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	configContainer.Internal.IsProduction = c.Bool("production")

	appService, err := applicationServiceCreator(kubernetesNamespace, configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	tag := "staging-" + kubernetesNamespace + "-latest"
	if kubernetesNamespace == loader.StagingEnvironment {
		tag = "staging-latest"
	}

	if configContainer.Internal.IsProduction == true {
		tag = "latest"
	}

	imagesService, err := serviceBuilder.GetImagesService()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	hasTag, err := imagesService.HasTag(configContainer.Cleanup, tag)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if !hasTag {
		return cli.NewExitError(fmt.Sprintf("No Image '%s' found for namespace '%s' ", tag, kubernetesNamespace), 0)
	}

	err = appService.Apply()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

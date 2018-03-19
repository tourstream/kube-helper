package app

import (
	"fmt"

	"kube-helper/util"

	"github.com/urfave/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CmdCleanUp(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	usedNamespaces := []string{"kube-system", "default"}

	for _, branchName := range branches {
		usedNamespaces = append(usedNamespaces, getNamespace(branchName, false, ""))
	}

	list, err := clientSet.CoreV1().Namespaces().List(v1.ListOptions{})

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	for _, namespace := range list.Items {
		if util.Contains(usedNamespaces, namespace.Name) {
			continue
		}

		appService, err := applicationServiceCreator(namespace.Name, configContainer)

		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		err = appService.DeleteByNamespace()

		if err != nil {
			fmt.Fprint(writer, err)
		}
	}

	return nil
}

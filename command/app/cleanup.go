package app

import (
	"fmt"

	"github.com/urfave/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"kube-helper/util"
)

func CmdCleanUp(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	if err != nil {
		return err
	}

	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return err
	}

	usedNamespaces := []string{}

	usedNamespaces = append(usedNamespaces, "kube-system", "default")

	for _, branchName := range branches {
		usedNamespaces = append(usedNamespaces, getNamespace(branchName, false))
	}

	list, err := clientSet.CoreV1().Namespaces().List(v1.ListOptions{})

	if err != nil {
		return err
	}

	for _, namespace := range list.Items {
		if util.InArray(usedNamespaces, namespace.Name) {
			continue
		}

		appService, err := serviceBuilder.GetApplicationService(clientSet, namespace.Name, configContainer)

		if err != nil {
			return err
		}

		err = appService.DeleteByNamespace()

		if err != nil {
			fmt.Fprint(writer, err)
		}
	}

	return nil
}

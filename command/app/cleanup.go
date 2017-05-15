package app

import (
	"kube-helper/util"

	"github.com/urfave/cli"
	"k8s.io/client-go/pkg/api/v1"
)

func CmdCleanUp(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, _ := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	branches, err := branchLoader.LoadBranches(configContainer.Bitbucket)

	if err != nil {
		return err
	}

	potentialNameSpaces := []string{}

	for _, branchName := range branches {
		potentialNameSpaces = append(potentialNameSpaces, getNamespace(branchName))
	}

	list, err := clientSet.CoreV1().Namespaces().List(v1.ListOptions{})

	if err != nil {
		return err
	}

	for _, namespace := range list.Items {
		if namespace.Name == "kube-system" || namespace.Name == "default" || util.InArray(potentialNameSpaces, namespace.Name) {
			continue
		}

		deleteApplicationByNamespace(clientSet, namespace.Name, configContainer)
	}

	return nil
}

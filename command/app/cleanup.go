package app

import (
	"github.com/urfave/cli"

	"k8s.io/client-go/pkg/api/v1"

	"kube-helper/util"
)

func CmdCleanUp(c *cli.Context) error {

	configContainer, _ := util.LoadConfigFromPath(c.String("config"))

	branches, err := util.GetBranches(configContainer.Bitbucket)

	potentialNameSpaces := []string{}

	for _, branchName := range branches {
		potentialNameSpaces = append(potentialNameSpaces, getNamespace(branchName))
	}

	createContainerService()
	createClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	list, err := clientset.CoreV1().Namespaces().List(v1.ListOptions{})
	util.CheckError(err)

	for _, namespace := range list.Items {
		if namespace.Name == "kube-system" || namespace.Name == "default" || util.InArray(potentialNameSpaces, namespace.Name) {
			continue
		}

		deleteApplicationByNamespace(namespace.Name, configContainer)
	}

	return nil
}

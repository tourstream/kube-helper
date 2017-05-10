package app

import (
	"fmt"


	"github.com/urfave/cli"
	"kube-helper/util"
)

func CmdHasNamespace(c *cli.Context) error {

	configContainer, _ := util.LoadConfigFromPath(c.String("config"))
	createContainerService()
	createClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	_, err := clientset.CoreV1().Namespaces().Get(getNamespace(c.Args().Get(0)))

	if err != nil {
		fmt.Println("false")
		return nil
	}

	fmt.Println("true")

	return nil
}

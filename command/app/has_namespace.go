package app

import (
	"fmt"

	"kube-helper/config"

	"github.com/urfave/cli"
)

func CmdHasNamespace(c *cli.Context) error {

	configContainer := config.LoadConfigFromPath(c.String("config"))
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

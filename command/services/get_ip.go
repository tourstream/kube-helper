package services

import (
	"github.com/urfave/cli"
	"fmt"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CmdGetIp(c *cli.Context) error {

	kubernetesNamespace := c.Args().Get(0)
	serviceName := c.Args().Get(1)

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}


	foundService, err := clientSet.CoreV1().Services(kubernetesNamespace).Get(serviceName, meta_v1.GetOptions{})

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	fmt.Fprintf(writer, "%s\n", foundService.Spec.ClusterIP)

	return nil
}

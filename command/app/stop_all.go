package app

import (
	"fmt"

	"github.com/urfave/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func CmdShutdownAll(c *cli.Context) error {

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	clientSet, err := serviceBuilder.GetClientSet(configContainer)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	list, err := clientSet.CoreV1().Namespaces().List(v1.ListOptions{})

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	for _, namespace := range list.Items {
		if strings.HasPrefix(namespace.Name, "kube") || namespace.Name == "default" || !strings.HasPrefix(namespace.Name, configContainer.Namespace.Prefix) {
			continue
		}

		appService, err := serviceBuilder.GetApplicationService(clientSet, strings.TrimPrefix(namespace.Name, configContainer.Namespace.Prefix + "-"), configContainer)

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

package app

import (
	"fmt"

	"kube-helper/util"

	"github.com/spf13/afero"
	"github.com/urfave/cli"
	"k8s.io/client-go/pkg/api/v1"
)

func CmdUpdate(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0))

	configContainer, _ := util.LoadConfigFromPath(c.String("config"))

	createUniveralDecoder()
	createContainerService()
	createClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	err := updateApplicationByNamespace(kubernetesNamespace, configContainer)

	util.CheckError(err)
	return nil
}

func updateApplicationByNamespace(kubernetesNamespace string, configContainer util.Config) error {
	err := isValidNamespace(kubernetesNamespace)

	if err != nil {
		return err
	}

	updateFromKubernetesConfig(kubernetesNamespace, configContainer.KubernetesConfigFilepath)

	pods, err := clientset.CoreV1().Pods(kubernetesNamespace).List(v1.ListOptions{})
	util.CheckError(err)
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	return nil
}

func updateFromKubernetesConfig(kubernetesNamespace string, path string) {
	util.ReplaceVariablesInFile(afero.NewOsFs(), path, func(splitLines []string) {
		updateKind(kubernetesNamespace, splitLines)
	})
}

package app

import (
	"fmt"

	"github.com/urfave/cli"
	"k8s.io/client-go/pkg/api/v1"
	"kube-helper/config"
	"kube-helper/util"
)

func CmdUpdate(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0))

	configContainer := config.LoadConfigFromPath(c.String("config"))

	createUniveralDecoder()
	createContainerService()
	createClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	err := updateApplicationByNamespace(kubernetesNamespace, configContainer)

	util.CheckError(err)
	return nil
}

func updateApplicationByNamespace(kubernetesNamespace string, configContainer config.Config) error {
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
	util.ReplaceVariablesInFile(path, tmpSplitFile, func() {
		updateKind(kubernetesNamespace)
	})
}

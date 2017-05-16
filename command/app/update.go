package app

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/spf13/afero"
	"github.com/urfave/cli"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/util/validation"
	"kube-helper/util"
	"k8s.io/client-go/pkg/runtime"
)


var universalDecoder runtime.Decoder

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

func createUniveralDecoder() {
	universalDecoder = api.Codecs.UniversalDecoder(unversioned.GroupVersion{
		Version: "v1",
	}, unversioned.GroupVersion{
		Group:   "extensions",
		Version: "v1beta1",
	}, unversioned.GroupVersion{
		Group:   "batch",
		Version: "v2alpha1",
	})
}

const namespaceNameFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"

var namespaceNameRegexp = regexp.MustCompile("^" + namespaceNameFmt + "$")

func isValidNamespace(kubernetesNamespace string) error {
	if !namespaceNameRegexp.MatchString(kubernetesNamespace) {
		return errors.New(validation.RegexError(namespaceNameFmt, "my-name", "123-abc"))
	}
	return nil
}

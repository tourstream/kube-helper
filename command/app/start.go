package app

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	"kube-helper/config"
	"kube-helper/util"

	"github.com/urfave/cli"
	"google.golang.org/api/dns/v1"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/util/validation"
)

var universalDecoder runtime.Decoder

func CmdStartUp(c *cli.Context) error {

	kubenetesNamespace := getNamespace(c.Args().Get(0))

	configContainer := config.LoadConfigFromPath(c.String("config"))

	createUniveralDecoder()
	createContainerService()
	createClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	err := createApplicationByNamespace(kubenetesNamespace, configContainer)

	util.CheckError(err)
	return nil
}

func createUniveralDecoder() {
	universalDecoder = api.Codecs.UniversalDecoder(unversioned.GroupVersion{
		Version: "v1",
	}, unversioned.GroupVersion{
		Group:   "extensions",
		Version: "v1beta1",
	})
}

func createApplicationByNamespace(kubenetesNamespace string, configContainer config.Config) error {
	err := isValidNamespace(kubenetesNamespace)

	if err != nil {
		return err
	}
	createNamespace(kubenetesNamespace)
	createFromKubernetesConfig(kubenetesNamespace, configContainer.KubernetesConfigFilepath)

	createDNSEntries(createDNSService(), kubenetesNamespace, getLoadBalancerIP(kubenetesNamespace, 60), configContainer.DNS)

	pods, err := clientset.CoreV1().Pods(kubenetesNamespace).List(v1.ListOptions{})
	util.CheckError(err)
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	return nil
}

func createFromKubernetesConfig(kubenetesNamespace string, path string) {
	util.ReplaceVariablesInFile(path, tmpSplitFile, func() {
		createKind(kubenetesNamespace)
	})
}

func createDNSEntries(dnsService *dns.Service, domainNamePart string, ip string, dnsConfig config.DNSConfig) {
	if ip == "" {
		log.Fatal("No Loadbalancer IP found.")
	}

	domain := domainNamePart + dnsConfig.DomainSuffix

	var cnames []string

	for _, cnameSuffix := range dnsConfig.CNameSuffix {
		cnames = append(cnames, domainNamePart+cnameSuffix)
	}

	createDNSEntry := &dns.Change{
		Additions: getResourceRecordSets(domain, cnames, ip),
	}

	_, err := dnsService.Changes.Create(dnsConfig.ProjectID, dnsConfig.ManagedZone, createDNSEntry).Do()
	util.CheckError(err)
	log.Printf("Created DNS Entries for %s", ip)
}

const namespaceNameFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"

var namespaceNameRegexp = regexp.MustCompile("^" + namespaceNameFmt + "$")

func isValidNamespace(kubernetesNamespace string) error {
	if !namespaceNameRegexp.MatchString(kubernetesNamespace) {
		return errors.New(validation.RegexError(namespaceNameFmt, "my-name", "123-abc"))
	}
	return nil
}

func createNamespace(kubenetesNamespace string) {
	if kubenetesNamespace != api.NamespaceDefault {
		_, err := clientset.CoreV1().Namespaces().Create(
			&v1.Namespace{
				ObjectMeta: v1.ObjectMeta{
					Name: kubenetesNamespace,
				},
			},
		)
		util.CheckError(err)

		log.Printf("Namespace \"%s\" was generated\n", kubenetesNamespace)
	} else {
		log.Fatalf("Namespace \"%s\" was already generated\n", kubenetesNamespace)
	}
}

package app

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/spf13/afero"
	"github.com/urfave/cli"
	"google.golang.org/api/dns/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/util/validation"
	"kube-helper/loader"
	"kube-helper/service"
)

var universalDecoder runtime.Decoder

func CmdStartUp(c *cli.Context) error {

	kubenetesNamespace := getNamespace(c.Args().Get(0))

	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, _ := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	return createApplicationByNamespace(clientSet, kubenetesNamespace, configContainer)
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

func createApplicationByNamespace(clientSet kubernetes.Interface, kubenetesNamespace string, configContainer loader.Config) error {
	err := isValidNamespace(kubenetesNamespace)

	if err != nil {
		return err
	}

	if hasNameSpace(clientSet, kubenetesNamespace) {
		fmt.Printf("Namespace \"%s\" was already generated\n", kubenetesNamespace)
		return nil
	}

	createNamespace(clientSet, kubenetesNamespace)
	err = createFromKubernetesConfig(clientSet, kubenetesNamespace, configContainer.KubernetesConfigFilepath)

	if err != nil {
		return err
	}

	ip, _ := getLoadBalancerIP(clientset, kubenetesNamespace, 60)

	dnsService, err := serviceBuilder.GetDNSService()

	if err != nil {
		return err
	}

	err = createDNSEntries(dnsService, kubenetesNamespace, ip, configContainer.DNS)

	if err != nil {
		return err
	}

	pods, err := clientSet.CoreV1().Pods(kubenetesNamespace).List(v1.ListOptions{})

	if err != nil {
		return err
	}

	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	return nil
}

func createFromKubernetesConfig(clientSet kubernetes.Interface, kubenetesNamespace string, path string) error {
	kindService := service.NewKind(clientSet)
	return loader.ReplaceVariablesInFile(afero.NewOsFs(), path, func(splitLines []string) error {
		return kindService.CreateKind(kubenetesNamespace, splitLines)
	})
}

func createDNSEntries(dnsService *dns.Service, domainNamePart string, ip string, dnsConfig loader.DNSConfig) error {
	if ip == "" {
		errors.New("No Loadbalancer IP found.")
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

	if err != nil {
		return err
	}

	log.Printf("Created DNS Entries for %s", ip)

	return nil
}

const namespaceNameFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"

var namespaceNameRegexp = regexp.MustCompile("^" + namespaceNameFmt + "$")

func isValidNamespace(kubernetesNamespace string) error {
	if !namespaceNameRegexp.MatchString(kubernetesNamespace) {
		return errors.New(validation.RegexError(namespaceNameFmt, "my-name", "123-abc"))
	}
	return nil
}

func createNamespace(clientSet kubernetes.Interface, kubenetesNamespace string) error {
	if kubenetesNamespace != api.NamespaceDefault {
		_, err := clientSet.CoreV1().Namespaces().Create(
			&v1.Namespace{
				ObjectMeta: v1.ObjectMeta{
					Name: kubenetesNamespace,
				},
			},
		)
		if err != nil {
			return err
		}

		log.Printf("Namespace \"%s\" was generated\n", kubenetesNamespace)

		return nil
	}
	return errors.New(fmt.Sprintf("Namespace \"%s\" was already generated\n", kubenetesNamespace))
}

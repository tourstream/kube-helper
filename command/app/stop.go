package app

import (
	"log"

	"github.com/urfave/cli"

	"google.golang.org/api/dns/v1"

	"k8s.io/client-go/pkg/api/v1"

	"kube-helper/config"
	"kube-helper/util"
)

func CmdShutdown(c *cli.Context) error {

	kubenetesNamespace := getNamespace(c.Args().Get(0))
	configContainer := config.LoadConfigFromPath(c.String("config"))

	createContainerService()
	createClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	deleteApplicationByNamespace(kubenetesNamespace, configContainer)

	return nil
}

func deleteApplicationByNamespace(kubenetesNamespace string, configContainer config.Config) {
	ip := getLoadBalancerIP(kubenetesNamespace, 10)

	deleteIngress(kubenetesNamespace, configContainer.ProjectID)
	deleteService(kubenetesNamespace)
	deleteDeployment(kubenetesNamespace)

	deleteNamespace(kubenetesNamespace)

	deleteDNSEntries(createDNSService(), kubenetesNamespace, ip, configContainer.DNS)
}

func deleteNamespace(kubenetesNamespace string) {
	err := clientset.CoreV1().Namespaces().Delete(kubenetesNamespace, &v1.DeleteOptions{})
	util.CheckError(err)

	log.Printf("Namespace \"%s\" was deleted\n", kubenetesNamespace)
}

func deleteDNSEntries(service *dns.Service, domainNamePart string, ip string, dnsConfig config.DNSConfig) {
	if ip == "" {
		return
	}

	domain := domainNamePart + dnsConfig.DomainSuffix

	var cnames []string

	for _, cnameSuffix := range dnsConfig.CNameSuffix {
		cnames = append(cnames, domainNamePart+cnameSuffix)
	}

	deleteDNSEntry := &dns.Change{
		Deletions: getResourceRecordSets(domain, cnames, ip),
	}

	_, err := service.Changes.Create(dnsConfig.ProjectID, dnsConfig.ManagedZone, deleteDNSEntry).Do()
	util.CheckError(err)

	log.Printf("Deleted DNS Entries for %s", ip)
}

func deleteIngress(kubenetesNamespace string, projectID string) {

	list, err := clientset.Ingresses(kubenetesNamespace).List(v1.ListOptions{})
	util.CheckError(err)

	err = clientset.Ingresses(kubenetesNamespace).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})
	util.CheckError(err)
	for _, ingress := range list.Items {
		addressName := ingress.Annotations["ingress.kubernetes.io/static-ip"]
		if len(addressName) > 0 {
			waitForStaticIPToBeDeleted(projectID, addressName, 60)
			log.Printf("%s is deleted and so the ingres with name \"%s\" is removed", addressName, ingress.Name)
		}
	}
}

func deleteDeployment(kubenetesNamespace string) {
	err := clientset.Deployments(kubenetesNamespace).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})

	util.CheckError(err)
}

func deleteService(kubenetesNamespace string) {

	list, err := clientset.Services(kubenetesNamespace).List(v1.ListOptions{})

	util.CheckError(err)

	for _, service := range list.Items {
		err = clientset.Services(kubenetesNamespace).Delete(service.Name, &v1.DeleteOptions{})

		util.CheckError(err)
	}
}

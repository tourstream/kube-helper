package app

import (
	"log"

	"github.com/urfave/cli"
	"google.golang.org/api/dns/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"kube-helper/loader"
)

func CmdShutdown(c *cli.Context) error {

	kubernetesNamespace := getNamespace(c.Args().Get(0))
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return err
	}

	clientSet, _ := serviceBuilder.GetClientSet(configContainer.ProjectID, configContainer.Zone, configContainer.ClusterID)

	return deleteApplicationByNamespace(clientSet, kubernetesNamespace, configContainer)
}

func deleteApplicationByNamespace(clientSet kubernetes.Interface, kubernetesNamespace string, configContainer loader.Config) error {
	ip, _ := getLoadBalancerIP(clientSet, kubernetesNamespace, 10)

	err := deleteIngress(clientSet, kubernetesNamespace, configContainer.ProjectID)

	if err != nil {
		return err
	}

	err = deleteService(clientSet, kubernetesNamespace)

	if err != nil {
		return err
	}

	err = deleteDeployment(clientSet, kubernetesNamespace)

	if err != nil {
		return err
	}

	err = deleteNamespace(clientSet, kubernetesNamespace)

	if err != nil {
		return err
	}

	log.Printf("Namespace \"%s\" was deleted\n", kubernetesNamespace)

	dnsService, err := serviceBuilder.GetDNSService()

	err = deleteDNSEntries(dnsService, kubernetesNamespace, ip, configContainer.DNS)

	if err != nil {
		return err
	}

	log.Printf("Deleted DNS Entries for %s", ip)

	return nil
}

func deleteNamespace(clientSet kubernetes.Interface, kubernetesNamespace string) error {
	return clientSet.CoreV1().Namespaces().Delete(kubernetesNamespace, &v1.DeleteOptions{})
}

func deleteDNSEntries(service *dns.Service, domainNamePart string, ip string, dnsConfig loader.DNSConfig) error {
	if ip == "" {
		return nil
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
	if err != nil {
		return err
	}

	return nil
}

func deleteIngress(clientSet kubernetes.Interface, kubernetesNamespace string, projectID string) error {

	list, err := clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).List(v1.ListOptions{})

	if err != nil {
		return err
	}

	err = clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})

	if err != nil {
		return err
	}

	for _, ingress := range list.Items {
		addressName := ingress.Annotations["ingress.kubernetes.io/static-ip"]
		if len(addressName) > 0 {
			err := waitForStaticIPToBeDeleted(projectID, addressName, 60)
			if err != nil {
				return err
			}
			log.Printf("%s is deleted and so the ingres with name \"%s\" is removed", addressName, ingress.Name)

		}
	}

	return nil
}

func deleteDeployment(clientSet kubernetes.Interface, kubernetesNamespace string) error {
	return clientSet.ExtensionsV1beta1().Deployments(kubernetesNamespace).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})

}

func deleteService(clientSet kubernetes.Interface, kubernetesNamespace string) error {

	list, err := clientSet.CoreV1().Services(kubernetesNamespace).List(v1.ListOptions{})

	if err != nil {
		return err
	}

	for _, service := range list.Items {
		err = clientSet.CoreV1().Services(kubernetesNamespace).Delete(service.Name, &v1.DeleteOptions{})

		if err != nil {
			return err
		}
	}

	return nil
}

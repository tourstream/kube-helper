package service

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"kube-helper/loader"

	"github.com/spf13/afero"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/util/validation"
)

type ApplicationServiceInterface interface {
	DeleteByNamespace() error
	CreateForNamespace() error
	UpdateByNamespace() error
	HasNamespace() bool
}

type applicationService struct {
	clientSet      kubernetes.Interface
	namespace      string
	config         loader.Config
	dnsService     *dns.Service
	computeService *compute.Service
}

func NewApplicationService(client kubernetes.Interface, namespace string, config loader.Config, dnsService *dns.Service, computeService *compute.Service) ApplicationServiceInterface {
	a := new(applicationService)
	a.clientSet = client
	a.namespace = namespace
	a.config = config
	a.dnsService = dnsService
	a.computeService = computeService
	return a
}

const namespaceNameFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"

var namespaceNameRegexp = regexp.MustCompile("^" + namespaceNameFmt + "$")

func (a *applicationService) CreateForNamespace() error {
	err := a.isValidNamespace()

	if err != nil {
		return err
	}

	if a.HasNamespace() {
		fmt.Printf("Namespace \"%s\" was already generated\n", a.namespace)
		return nil
	}

	a.createNamespace()
	err = a.createFromKubernetesConfig()

	if err != nil {
		return err
	}

	ip, err := a.getLoadBalancerIP(60)

	if err != nil {
		return err
	}

	err = a.createDNSEntries(ip, a.config.DNS)

	if err != nil {
		return err
	}

	pods, err := a.clientSet.CoreV1().Pods(a.namespace).List(v1.ListOptions{})

	if err != nil {
		return err
	}

	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	return nil
}

func (a *applicationService) DeleteByNamespace() error {
	ip, _ := a.getLoadBalancerIP(10)

	err := a.deleteIngress(a.config.ProjectID)

	if err != nil {
		return err
	}

	err = a.deleteService()

	if err != nil {
		return err
	}

	err = a.deleteDeployment()

	if err != nil {
		return err
	}

	err = a.deleteNamespace()

	if err != nil {
		return err
	}

	log.Printf("Namespace \"%s\" was deleted\n", a.namespace)

	err = a.deleteDNSEntries(ip, a.config.DNS)

	if err != nil {
		return err
	}

	log.Printf("Deleted DNS Entries for %s", ip)

	return nil
}

func (a *applicationService) UpdateByNamespace() error {
	err := a.isValidNamespace()

	if err != nil {
		return err
	}

	err = a.updateFromKubernetesConfig()

	if err != nil {
		return err
	}

	pods, err := a.clientSet.CoreV1().Pods(a.namespace).List(v1.ListOptions{})

	if err != nil {
		return err
	}

	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	return nil
}

func (a *applicationService) HasNamespace() bool {
	_, err := a.clientSet.CoreV1().Namespaces().Get(a.namespace)

	if err != nil {
		return false
	}

	return true
}

func (a *applicationService) deleteNamespace() error {
	return a.clientSet.CoreV1().Namespaces().Delete(a.namespace, &v1.DeleteOptions{})
}

func (a *applicationService) createDNSEntries(ip string, dnsConfig loader.DNSConfig) error {
	if ip == "" {
		return errors.New("No Loadbalancer IP found.")
	}

	domain := a.namespace + dnsConfig.DomainSuffix

	var cnames []string

	for _, cnameSuffix := range dnsConfig.CNameSuffix {
		cnames = append(cnames, a.namespace+cnameSuffix)
	}

	createDNSEntry := &dns.Change{
		Additions: a.getResourceRecordSets(domain, cnames, ip),
	}

	_, err := a.dnsService.Changes.Create(dnsConfig.ProjectID, dnsConfig.ManagedZone, createDNSEntry).Do()

	if err != nil {
		return err
	}

	log.Printf("Created DNS Entries for %s", ip)

	return nil
}

func (a *applicationService) deleteDNSEntries(ip string, dnsConfig loader.DNSConfig) error {
	if ip == "" {
		return nil
	}

	domain := a.namespace + dnsConfig.DomainSuffix

	var cnames []string

	for _, cnameSuffix := range dnsConfig.CNameSuffix {
		cnames = append(cnames, a.namespace+cnameSuffix)
	}

	deleteDNSEntry := &dns.Change{
		Deletions: a.getResourceRecordSets(domain, cnames, ip),
	}

	_, err := a.dnsService.Changes.Create(dnsConfig.ProjectID, dnsConfig.ManagedZone, deleteDNSEntry).Do()
	if err != nil {
		return err
	}

	return nil
}

func (a *applicationService) deleteIngress(projectID string) error {

	list, err := a.clientSet.ExtensionsV1beta1().Ingresses(a.namespace).List(v1.ListOptions{})

	if err != nil {
		return err
	}

	err = a.clientSet.ExtensionsV1beta1().Ingresses(a.namespace).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})

	if err != nil {
		return err
	}

	for _, ingress := range list.Items {
		addressName := ingress.Annotations["ingress.kubernetes.io/static-ip"]
		if len(addressName) > 0 {
			err := a.waitForStaticIPToBeDeleted(projectID, addressName, 60)
			if err != nil {
				return err
			}
			log.Printf("%s is deleted and so the ingres with name \"%s\" is removed", addressName, ingress.Name)

		}
	}

	return nil
}

func (a *applicationService) deleteDeployment() error {
	return a.clientSet.ExtensionsV1beta1().Deployments(a.namespace).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})

}

func (a *applicationService) deleteService() error {

	list, err := a.clientSet.CoreV1().Services(a.namespace).List(v1.ListOptions{})

	if err != nil {
		return err
	}

	for _, service := range list.Items {
		err = a.clientSet.CoreV1().Services(a.namespace).Delete(service.Name, &v1.DeleteOptions{})

		if err != nil {
			return err
		}
	}

	return nil
}

func (a *applicationService) getLoadBalancerIP(maxRetries int) (string, error) {
	var ip string

	for retries := 0; retries < maxRetries; retries++ {
		loadbalancer, err := a.clientSet.ExtensionsV1beta1().Ingresses(a.namespace).Get("loadbalancer")

		if err != nil {
			return "", err
		}

		if len(loadbalancer.Status.LoadBalancer.Ingress) > 0 {
			ip = loadbalancer.Status.LoadBalancer.Ingress[0].IP
			break
		}
		log.Print("Waiting for Loadbalancer IP")
		time.Sleep(time.Second * 5)
	}
	if ip == "" {
		return "", errors.New("No Loadbalancer IP found")
	}
	log.Printf("Loadbalancer IP : %s", ip)

	return ip, nil
}

func (a *applicationService) getResourceRecordSets(domain string, cnames []string, ip string) []*dns.ResourceRecordSet {
	recordSet := []*dns.ResourceRecordSet{
		{
			Rrdatas: []string{
				ip,
			},
			Ttl:  300,
			Type: "A",
			Name: domain,
		},
	}

	for _, cname := range cnames {
		recordSet = append(recordSet, &dns.ResourceRecordSet{
			Rrdatas: []string{
				domain,
			},
			Ttl:  300,
			Type: "CNAME",
			Name: cname,
		})
	}

	return recordSet
}

func (a *applicationService) waitForStaticIPToBeDeleted(projectID string, addressName string, maxRetries int) error {

	addressList, err := a.computeService.GlobalAddresses.List(projectID).Do()

	if err != nil {
		return err
	}

	for _, address := range addressList.Items {
		if address.Name == addressName {
			for retries := 0; retries < maxRetries; retries++ {
				_, err := a.computeService.GlobalAddresses.Get(projectID, address.Name).Do()
				if err != nil {
					break
				}
				log.Printf("Waiting for IP \"%s\" to be released", address.Name)
				time.Sleep(time.Second * 5)
			}
		}
	}

	return nil
}

func (a *applicationService) isValidNamespace() error {
	if !namespaceNameRegexp.MatchString(a.namespace) {
		return errors.New(validation.RegexError(namespaceNameFmt, "my-name", "123-abc"))
	}
	return nil
}

func (a *applicationService) createNamespace() error {
	if a.namespace != api.NamespaceDefault {
		_, err := a.clientSet.CoreV1().Namespaces().Create(
			&v1.Namespace{
				ObjectMeta: v1.ObjectMeta{
					Name: a.namespace,
				},
			},
		)
		if err != nil {
			return err
		}

		log.Printf("Namespace \"%s\" was generated\n", a.namespace)

		return nil
	}
	return errors.New(fmt.Sprintf("Namespace \"%s\" was already generated\n", a.namespace))
}

func (a *applicationService) createFromKubernetesConfig() error {
	kindService := NewKind(a.clientSet)
	return loader.ReplaceVariablesInFile(afero.NewOsFs(), a.config.KubernetesConfigFilepath, func(splitLines []string) error {
		return kindService.CreateKind(a.namespace, splitLines)
	})
}

func (a *applicationService) updateFromKubernetesConfig() error {
	kindService := NewKind(a.clientSet)
	return loader.ReplaceVariablesInFile(afero.NewOsFs(), a.config.KubernetesConfigFilepath, func(splitLines []string) error {
		return kindService.UpdateKind(a.namespace, splitLines)
	})
}

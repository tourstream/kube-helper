package app

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"kube-helper/loader"
	"os"
	"strings"

	"io"

	"kube-helper/service/builder"

	"kube-helper/service/kind"

	"kube-helper/service/app/watcher"

	"github.com/spf13/afero"
	compute_v1 "google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/servicemanagement/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilClock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var serviceBuilder = builder.NewServiceBuilder()
var clock utilClock.Clock = new(utilClock.RealClock)
var replaceVariablesInFile loader.ReplaceFunc = loader.ReplaceVariablesInFile
var writer io.Writer = os.Stdout
var kindServiceCreator = kind.NewKind
var watchControllerCreator = watcher.New

type ApplicationServiceInterface interface {
	DeleteByNamespace() error
	Apply() error
	HasNamespace() bool
	GetDomain(dnsConfig loader.DNSConfig) string
	HandleIngressAnnotationOnApply() error
}

type applicationService struct {
	clientSet         kubernetes.Interface
	prefixedNamespace string
	namespace         string
	config            loader.Config
	dnsService        *dns.Service
	computeService    *compute_v1.Service
	serviceManagement *servicemanagement.APIService
}

func NewApplicationService(namespace string, config loader.Config) (ApplicationServiceInterface, error) {
	a := new(applicationService)

	clientSet, err := serviceBuilder.GetClientSet(config)

	if err != nil {
		return nil, err
	}

	dnsService, err := serviceBuilder.GetDNSService()

	if err != nil {
		return nil, err
	}

	computeService, err := serviceBuilder.GetComputeService()

	if err != nil {
		return nil, err
	}

	serviceManagement, err := serviceBuilder.GetServiceManagementService()

	if err != nil {
		return nil, err
	}

	a.clientSet = clientSet
	a.prefixedNamespace = namespace
	if config.Namespace.Prefix != "" {
		a.prefixedNamespace = config.Namespace.Prefix + "-" + namespace
	}
	a.namespace = namespace
	a.config = config
	a.dnsService = dnsService
	a.computeService = computeService
	a.serviceManagement = serviceManagement
	return a, nil
}

const namespaceNameFmt = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"

var namespaceNameRegexp = regexp.MustCompile("^" + namespaceNameFmt + "$")

func (a *applicationService) Apply() error {
	err := a.isValidNamespace()

	if err != nil {
		return err
	}

	update := true

	if !a.HasNamespace() {
		update = false
		err = a.createNamespace()
		if err != nil {
			return err
		}
	}

	if a.config.Endpoints.Enabled {
		err = a.setEndpointEnvVariables()
		if err != nil {
			return err
		}
	}

	ctrl, ipCh, stopCh := a.waitLoadBalancerIP()

	defer close(stopCh)
	defer close(ipCh)
	go ctrl.Run(stopCh)

	err = a.applyFromConfig()

	if err != nil {
		return err
	}

	if !update && a.config.Cluster.Type == "gcp" {

		ip := <-ipCh

		err = a.createDNSEntries(ip, a.config.DNS)

		if err != nil {
			return err
		}
	}

	err = a.HandleIngressAnnotationOnApply()

	if err != nil {
		return err
	}

	pods, err := a.clientSet.CoreV1().Pods(a.prefixedNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	fmt.Fprintf(writer, "There are %d pods in the cluster\n", len(pods.Items))

	return nil
}

func (a *applicationService) DeleteByNamespace() error {

	nsObject, err := a.clientSet.CoreV1().Namespaces().Get(a.prefixedNamespace, meta_v1.GetOptions{})

	if err != nil {
		return err
	}

	var ip string

	if ingressExists, ok := nsObject.Annotations[kind.IngressKey]; ok && ingressExists == kind.IngressExists {

		ctrl, ipCh, stopCh := a.waitLoadBalancerIP()

		defer close(stopCh)
		defer close(ipCh)
		go ctrl.Run(stopCh)

		ip = <-ipCh

		err := a.deleteIngress(a.config.Cluster.ProjectID)

		if err != nil {
			return err
		}
	}

	err = a.deleteNamespace()

	if err != nil {
		return err
	}

	fmt.Fprintf(writer, "Namespace \"%s\" was deleted\n", a.prefixedNamespace)

	err = a.deleteDNSEntries(ip, a.config.DNS)

	if err != nil {
		return err
	}

	return nil
}

func (a *applicationService) HasNamespace() bool {
	_, err := a.clientSet.CoreV1().Namespaces().Get(a.prefixedNamespace, meta_v1.GetOptions{})

	if err != nil {
		return false
	}

	return true
}

func (a *applicationService) GetDomain(dnsConfig loader.DNSConfig) string {
	if a.namespace == loader.ProductionEnvironment {
		return dnsConfig.BaseDomain
	}

	if dnsConfig.BaseDomain != "" {
		return a.namespace + dnsConfig.DomainSpacer + dnsConfig.BaseDomain
	}

	return a.namespace + dnsConfig.DomainSuffix
}

func (a *applicationService) setEndpointEnvVariables() error {

	domain := strings.TrimSuffix(a.GetDomain(a.config.DNS), ".")

	configs, err := a.serviceManagement.Services.Configs.List(domain).Do()

	if err != nil {
		return err
	}

	err = os.Setenv("ENDPOINT_VERSION", configs.ServiceConfigs[0].Id)

	if err != nil {
		return err
	}

	return os.Setenv("ENDPOINT_DOMAIN", domain)
}

func (a *applicationService) deleteNamespace() error {
	return a.clientSet.CoreV1().Namespaces().Delete(a.prefixedNamespace, &meta_v1.DeleteOptions{})
}

func (a *applicationService) createDNSEntries(ip string, dnsConfig loader.DNSConfig) error {
	var cnames []string

	for _, cnameSuffix := range dnsConfig.CNameSuffix {
		cnames = append(cnames, a.namespace+cnameSuffix)
	}

	createDNSEntry := &dns.Change{
		Additions: a.getResourceRecordSets(a.GetDomain(dnsConfig), cnames, ip),
	}

	_, err := a.dnsService.Changes.Create(dnsConfig.ProjectID, dnsConfig.ManagedZone, createDNSEntry).Do()

	if err != nil {
		return err
	}

	fmt.Fprintf(writer, "Created DNS Entries for %s\n", ip)

	return nil
}

func (a *applicationService) deleteDNSEntries(ip string, dnsConfig loader.DNSConfig) error {
	if ip == "" {
		return nil
	}

	var cnames []string

	for _, cnameSuffix := range dnsConfig.CNameSuffix {
		cnames = append(cnames, a.namespace+cnameSuffix)
	}

	deleteDNSEntry := &dns.Change{
		Deletions: a.getResourceRecordSets(a.GetDomain(dnsConfig), cnames, ip),
	}

	_, err := a.dnsService.Changes.Create(dnsConfig.ProjectID, dnsConfig.ManagedZone, deleteDNSEntry).Do()
	if err != nil {
		return err
	}
	fmt.Fprintf(writer, "Deleted DNS Entries for %s", ip)
	return nil
}

func (a *applicationService) deleteIngress(projectID string) error {

	list, err := a.clientSet.ExtensionsV1beta1().Ingresses(a.prefixedNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	err = a.clientSet.ExtensionsV1beta1().Ingresses(a.prefixedNamespace).DeleteCollection(&meta_v1.DeleteOptions{}, meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	if projectID == "" {
		return nil
	}

	for _, ingress := range list.Items {
		if addressName, ok := ingress.Annotations["ingress.kubernetes.io/static-ip"]; ok && addressName != "" {
			err := a.waitForStaticIPToBeDeleted(projectID, addressName, 60)
			if err != nil {
				return err
			}
			fmt.Fprintf(writer, "%s is deleted and so the ingres with name \"%s\" is removed\n", addressName, ingress.Name)

		}
	}

	return nil
}

func (a *applicationService) waitLoadBalancerIP() (cache.Controller, chan string, chan struct{}) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return a.clientSet.ExtensionsV1beta1().Ingresses(a.prefixedNamespace).List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return a.clientSet.ExtensionsV1beta1().Ingresses(a.prefixedNamespace).Watch(options)
			},
		},
		&v1beta1.Ingress{},
		0, // Skip resync
		cache.Indexers{},
	)

	stopCh := make(chan struct{})
	ipCh := make(chan string, 1)

	processWatch := func(obj interface{}) {

		if obj == nil {
			return
		}

		if ingressType, ok := obj.(*v1beta1.Ingress).Annotations["kubernetes.io/ingress.class"]; ok && ingressType == "gce" && len(obj.(*v1beta1.Ingress).Status.LoadBalancer.Ingress) > 0 {
			ip := obj.(*v1beta1.Ingress).Status.LoadBalancer.Ingress[0].IP
			ipCh <- ip
		}
	}

	return watchControllerCreator(a.clientSet, informer, processWatch), ipCh, stopCh
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
				fmt.Fprintf(writer, "Waiting for IP \"%s\" to be released\n", address.Name)
				clock.Sleep(time.Second * 5)
			}
		}
	}

	return nil
}

func (a *applicationService) isValidNamespace() error {
	if !namespaceNameRegexp.MatchString(a.prefixedNamespace) {
		return errors.New(validation.RegexError(namespaceNameFmt, "my-name", "123-abc"))
	}
	return nil
}

func (a *applicationService) createNamespace() error {
	_, err := a.clientSet.CoreV1().Namespaces().Create(
		&v1.Namespace{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: a.prefixedNamespace,
			},
		},
	)
	if err != nil {
		return err
	}

	fmt.Fprintf(writer, "Namespace \"%s\" was generated\n", a.prefixedNamespace)

	return nil
}

func (a *applicationService) applyFromConfig() error {

	imageService, err := serviceBuilder.GetImagesService()

	if err != nil {
		return err
	}

	kindService := kindServiceCreator(a.clientSet, imageService, a.config)

	err = replaceVariablesInFile(afero.NewOsFs(), a.config.KubernetesConfigFilepath, func(splitLines []string) error {
		return kindService.ApplyKind(a.prefixedNamespace, splitLines, a.namespace)
	})

	if err != nil {
		return err
	}

	return kindService.CleanupKind(a.prefixedNamespace)
}

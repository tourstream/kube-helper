package service

import (
	"strings"
	"google.golang.org/api/compute/v1"
	"fmt"
	"errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/extensions/v1beta1"
)

type NamedAddress struct {
	ipName string
	port   string
}

var projectId string
var kb8Namespace string


func (a *applicationService) HandleIngressAnnotationOnApply() error {
	projectId = a.config.Cluster.ProjectID
	kb8Namespace = a.prefixedNamespace

	annotations, err := a.getIngressAnnotations()

	if err != nil {
		return err
	}

	if annotations == nil {
		fmt.Fprint(writer, "No Annotations to process")
		return nil
	}

	for annotationName, annotationValue := range annotations {
		err := a.applyAnnotation(annotationName, annotationValue)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *applicationService) getIngressAnnotations() (map[string]string, error) {
	ingressList, err := a.clientSet.ExtensionsV1beta1().Ingresses(a.prefixedNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return nil, err
	}

	ingress, err := a.getIngress(ingressList)

	if err != nil {
		fmt.Fprintf(writer, "%s\n", err.Error())
		return nil, nil
	}

	return ingress.Annotations, nil
}

func (a *applicationService) getIngress(ingressList *v1beta1.IngressList) (v1beta1.Ingress, error) {
	for _, ingress := range ingressList.Items {
		if ingress.Name != "" && ingress.Annotations != nil && len(ingress.Annotations) > 0 {
			return ingress, nil
		}
	}

	return v1beta1.Ingress{}, errors.New("No suitable ingress found")
}

func (a *applicationService) applyAnnotation(annotationName string, annotationValue string) error {
	if !strings.Contains(annotationName, "tourstream.eu/") {
		return nil
	}

	switch annotationName {
	case "tourstream.eu/named-ip-address":
		return a.addNewRulesToLoadBalancer(annotationValue)
	case "tourstream.eu/pre-shared-cert":
		return a.addCertificatesToHttpsProxies(strings.Split(annotationValue, ","))
	default:
		return errors.New("not supported Annotation")
	}
}

func (a *applicationService) addCertificatesToHttpsProxies(certificates []string) error {

	certificateList, err := a.getCertificateList(certificates)
	if err != nil {
		return err
	}

	proxies, err := a.getProxyList()
	if err != nil {
		return err
	}

	for _, proxy := range proxies {
		err = a.appendCertificates(proxy, certificateList)

		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Certificates %s added to %s\n", certificateList, proxy.Name)
	}

	return nil
}

func (a *applicationService) getProxyList() ([]*compute.TargetHttpsProxy, error) {
	proxies, err := a.computeService.TargetHttpsProxies.List(projectId).Do()
	if err != nil {
		return nil, err
	}

	var proxyList []*compute.TargetHttpsProxy
	for _, proxy := range proxies.Items {
		if strings.Contains(proxy.Name, kb8Namespace) {
			proxyList = append(proxyList, proxy)
		}
	}

	return proxyList, nil
}

func (a *applicationService) getCertificateList(certificates []string) ([]string, error) {
	var certificateList []string

	for _, certificate := range certificates {
		if certificate == "" {
			continue
		}

		item, err := a.getCertificateLink(certificate)

		if err != nil {
			return nil, err
		}

		certificateList = append(certificateList, item)
	}

	if len(certificateList) == 0 {
		return nil, errors.New("No Certificate to add")
	}

	return certificateList, nil
}

func (a *applicationService) appendCertificates(proxy *compute.TargetHttpsProxy, certificateList []string) error {
	for _, cert := range certificateList {
		proxy.SslCertificates = a.addToList(proxy.SslCertificates, cert)
	}

	certificatesRequest := new(compute.TargetHttpsProxiesSetSslCertificatesRequest)
	certificatesRequest.SslCertificates = proxy.SslCertificates
	_, err := a.computeService.TargetHttpsProxies.SetSslCertificates(projectId, proxy.Name, certificatesRequest).Do()

	return err
}

func (a *applicationService) addToList(list []string, value string) ([]string) {
	for _, entry := range list {
		if entry == value {
			return list
		}
	}

	list = append(list, value)

	return list
}

func (a *applicationService) getCertificateLink(certificateName string) (string, error) {
	certificate, err := a.computeService.SslCertificates.Get(projectId, certificateName).Do()

	if err != nil {
		return "", err
	}

	return certificate.SelfLink, nil
}

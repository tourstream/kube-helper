package services

import (
	"github.com/urfave/cli"
	"kube-helper/service"
	"google.golang.org/api/compute/v1"
	"strings"
)

func CmdAddCertificate(c *cli.Context) error {
	return runCmd(c, addToList)
}

func CmdRemoveCertificate(c *cli.Context) error {
	return runCmd(c, removeFromList)
}

func runCmd(c *cli.Context, listFunction func(list []string, value string) ([]string)) error {
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	projectId = configContainer.Cluster.ProjectID
	if projectId == "" {
		return cli.NewExitError("projectId is empty", 1)
	}

	kubernetesNamespace := configContainer.Namespace.Prefix + "-" + c.Args().Get(0)
	certificateName := c.String("ssl-certificate")

	if certificateName == "" {
		return cli.NewExitError("ssl-certificate not set", 1)
	}

	builder := new(service.Builder)
	computeService, err := builder.GetComputeService()

	certificate, err := computeService.SslCertificates.Get("e-tourism-suite", certificateName).Do()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	proxy, err := getHttpsProxy(computeService, kubernetesNamespace)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	newList := listFunction(proxy.SslCertificates, certificate.SelfLink)
	certificatesRequest := new(compute.TargetHttpsProxiesSetSslCertificatesRequest)
	certificatesRequest.SslCertificates = newList

	_, err = computeService.TargetHttpsProxies.SetSslCertificates("e-tourism-suite", proxy.Name, certificatesRequest).Do()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

func getHttpsProxy(computeService *compute.Service, kubernetesNamespace string) (*compute.TargetHttpsProxy, error) {

	proxies, err := computeService.TargetHttpsProxies.List(projectId).Do()
	if err != nil {
		return nil, cli.NewExitError(err.Error(), 1)
	}

	for _, proxy := range proxies.Items {
		if strings.Contains(proxy.Name, kubernetesNamespace) {
			return proxy, nil
		}
	}

	return nil, cli.NewExitError("HttpsProxy not found", 1)
}

func addToList(list []string, value string) ([]string) {
	inList := false

	for _, entry := range list {
		if entry == value {
			inList = true
			break
		}
	}

	if !inList {
		list = append(list, value)
	}

	return list
}

func removeFromList(list []string, value string) ([]string) {
	var newList []string

	for _, entry := range list {
		if entry != value {
			newList = append(newList, entry)
		}
	}

	return newList
}

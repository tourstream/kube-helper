package service

import (
	"encoding/base64"

	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type BuilderInterface interface {
	GetClientSet(projectID string, zone string, clusterId string) (kubernetes.Interface, error)
	GetDNSService() (*dns.Service, error)
	GetSqlService() (*sqladmin.Service, error)
	GetStorageService() (*storage.Service, error)
	GetClient(scope ...string) (*http.Client, error)
}

type Builder struct {
}

func (h *Builder) GetClientSet(projectID string, zone string, clusterId string) (kubernetes.Interface, error) {
	cService, err := h.getContainerService()

	if err != nil {
		return nil, err
	}

	cluster, err := cService.Projects.Zones.Clusters.Get(projectID, zone, clusterId).Do()
	if err != nil {
		return nil, err
	}

	kubernetesConfig := &rest.Config{
		Host: "https://" + cluster.Endpoint,
		AuthProvider: &clientcmdapi.AuthProviderConfig{
			Name: "gcp",
		},
	}

	ca, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)

	if err != nil {
		return nil, err
	}

	kubernetesConfig.TLSClientConfig.CAData = ca

	return kubernetes.NewForConfig(kubernetesConfig)
}

func (h *Builder) getContainerService() (*container.Service, error) {
	client, err := h.GetClient(container.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return container.New(client)
}

func (h *Builder) GetDNSService() (*dns.Service, error) {
	client, err := h.GetClient(dns.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return dns.New(client)
}

func (h *Builder) GetSqlService() (*sqladmin.Service, error) {
	client, err := h.GetClient(sqladmin.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return sqladmin.New(client)
}

func (h *Builder) GetStorageService() (*storage.Service, error) {
	client, err := h.GetClient(storage.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return storage.New(client)
}

func (h *Builder) GetClient(scope ...string) (*http.Client, error) {
	ctx := context.Background()

	return google.DefaultClient(ctx, scope...)
}

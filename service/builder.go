package service

import (
	"encoding/base64"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type BuilderInterface interface {
	GetClientSet(projectID string, zone string, clusterId string) (kubernetes.Interface, error)
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

	client, err := google.DefaultClient(context.Background(), container.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return container.New(client)
}

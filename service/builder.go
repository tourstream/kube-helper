package service

import (
	"encoding/base64"
	"net/http"

	"errors"
	"kube-helper/loader"

	StorageClient "cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/servicemanagement/v1"
	"google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type BuilderInterface interface {
	GetClientSet(config loader.Config) (kubernetes.Interface, error)
	GetDNSService() (*dns.Service, error)
	GetSqlService() (*sqladmin.Service, error)
	GetStorageService(bucket string) (BucketServiceInterface, error)
	GetClient(scope ...string) (*http.Client, error)
	GetApplicationService(client kubernetes.Interface, namespace string, config loader.Config) (ApplicationServiceInterface, error)
	GetImagesService() (ImagesInterface, error)
	GetKindService(client kubernetes.Interface, imagesService ImagesInterface, config loader.Config) KindInterface
}

type Builder struct {
}

func (h *Builder) GetClientSet(config loader.Config) (kubernetes.Interface, error) {
	cService, err := h.getContainerService()

	if err != nil {
		return nil, err
	}

	switch clusterType := config.Cluster.Type; clusterType {
	case "local":
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err := kubeConfig.ClientConfig()
		if err != nil {
			return nil, errors.New("Failed loading client config")
		}
		return kubernetes.NewForConfig(config)
	case "gcp":
		fallthrough
	default:
		cluster, err := cService.Projects.Zones.Clusters.Get(config.Cluster.ProjectID, config.Cluster.Zone, config.Cluster.ClusterID).Do()
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

func (h *Builder) GetStorageService(bucket string) (BucketServiceInterface, error) {
	httpClient, err := h.GetClient(storage.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	storageService, err := storage.New(httpClient)

	if err != nil {
		return nil, err
	}

	storageClient, err := h.getStorageClient()

	if err != nil {
		return nil, err
	}

	return NewBucketService(bucket, httpClient, storageService, storageClient), nil
}

func (h *Builder) GetApplicationService(client kubernetes.Interface, namespace string, config loader.Config) (ApplicationServiceInterface, error) {

	dnsService, err := h.GetDNSService()

	if err != nil {
		return nil, err
	}

	computeService, err := h.getComputeService()

	if err != nil {
		return nil, err
	}

	serviceManagementService, err := h.getServiceManagementService()

	if err != nil {
		return nil, err
	}

	return NewApplicationService(client, namespace, config, dnsService, computeService, serviceManagementService), nil
}

func (h *Builder) GetClient(scope ...string) (*http.Client, error) {
	ctx := context.Background()

	return google.DefaultClient(ctx, scope...)
}

func (h *Builder) getStorageClient() (*StorageClient.Client, error) {
	return StorageClient.NewClient(context.Background())
}

func (h *Builder) getComputeService() (*compute.Service, error) {
	httpClient, err := h.GetClient(compute.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return compute.New(httpClient)
}

func (h *Builder) getServiceManagementService() (*servicemanagement.APIService, error) {
	httpClient, err := h.GetClient(compute.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return servicemanagement.New(httpClient)
}

func (h *Builder) GetImagesService() (ImagesInterface, error) {
	imageService, err := newImagesService()

	if err != nil {
		return nil, err
	}

	return imageService, nil
}

func (h *Builder) GetKindService(client kubernetes.Interface, imagesService ImagesInterface, config loader.Config) KindInterface {
	kindService := newKind(client, imagesService, config)

	return kindService
}

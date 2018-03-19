package builder

import (
	"encoding/base64"
	"net/http"

	"errors"
	"kube-helper/loader"

	"kube-helper/service/bucket"
	"kube-helper/service/image"

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

// ServiceBuilder API
type ServiceBuilderInterface interface {
	GetClientSet(config loader.Config) (kubernetes.Interface, error)
	GetDNSService() (*dns.Service, error)
	GetSQLService() (*sqladmin.Service, error)
	GetStorageService(bucketName string) (bucket.BucketServiceInterface, error)
	GetImagesService() (image.ImagesInterface, error)
	GetComputeService() (*compute.Service, error)
	GetServiceManagementService() (*servicemanagement.APIService, error)
}

type builder struct{}

// NewServiceBuilder is the constructor method and returns a service which implements ServiceBuilderInterface
func NewServiceBuilder() ServiceBuilderInterface {
	return &builder{}
}

// GetClientSet returns the kubernetes client for local or gcp
func (h *builder) GetClientSet(config loader.Config) (kubernetes.Interface, error) {

	switch clusterType := config.Cluster.Type; clusterType {
	case "local":
		return h.getClientSetForLocal()
	case "gcp":
		fallthrough
	default:
		return h.getClientSetForGoogleCloudPlatform(config)
	}
}

func (h *builder) getClientSetForGoogleCloudPlatform(config loader.Config) (kubernetes.Interface, error) {

	cService, err := h.getContainerService()

	if err != nil {
		return nil, err
	}

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

func (h *builder) getClientSetForLocal() (kubernetes.Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.New("failed loading client config")
	}
	return kubernetes.NewForConfig(config)
}

func (h *builder) getContainerService() (*container.Service, error) {
	client, err := h.getClient(container.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return container.New(client)
}

// GetDNSService returns the DNS API Client for gcp
func (h *builder) GetDNSService() (*dns.Service, error) {
	client, err := h.getClient(dns.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return dns.New(client)
}

// GetSQLService returns the SQL API Client for gcp
func (h *builder) GetSQLService() (*sqladmin.Service, error) {
	client, err := h.getClient(sqladmin.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return sqladmin.New(client)
}

// GetStorageService returns the Storage API Client for gcp
func (h *builder) GetStorageService(bucketName string) (bucket.BucketServiceInterface, error) {
	httpClient, err := h.getClient(storage.CloudPlatformScope)

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

	return bucket.NewBucketService(bucketName, httpClient, storageService, storageClient), nil
}

func (h *builder) getClient(scope ...string) (*http.Client, error) {
	ctx := context.Background()

	return google.DefaultClient(ctx, scope...)
}

func (h *builder) getStorageClient() (*StorageClient.Client, error) {
	return StorageClient.NewClient(context.Background())
}

// GetComputeService returns the Compute API Client for gcp
func (h *builder) GetComputeService() (*compute.Service, error) {
	httpClient, err := h.getClient(compute.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return compute.New(httpClient)
}

// GetServiceManagementService returns the Service Management API Client for gcp
func (h *builder) GetServiceManagementService() (*servicemanagement.APIService, error) {
	httpClient, err := h.getClient(compute.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	return servicemanagement.New(httpClient)
}

// GetImagesService returns the Image Service
func (h *builder) GetImagesService() (image.ImagesInterface, error) {
	return image.NewImagesService()
}

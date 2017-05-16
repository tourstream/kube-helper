package app

import (
	"encoding/base64"
	"errors"
	"log"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/dns/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"kube-helper/util"
)

var containerService *container.Service
var clientset *kubernetes.Clientset

const stagingEnvironment = "staging"

var clientSetCreator = kubernetes.NewForConfig

func createContainerService() {
	ctx := context.Background()

	client, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	util.CheckError(err)
	containerService, err = container.New(client)
	util.CheckError(err)
}

func createClientSet(projectID string, zone string, clusterId string) {
	cluster, err := containerService.Projects.Zones.Clusters.Get(projectID, zone, clusterId).Do()
	util.CheckError(err)

	kubernetesConfig := &rest.Config{
		Host: "https://" + cluster.Endpoint,
		AuthProvider: &clientcmdapi.AuthProviderConfig{
			Name: "gcp",
		},
	}

	ca, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)

	util.CheckError(err)

	kubernetesConfig.TLSClientConfig.CAData = ca

	clientset, err = clientSetCreator(kubernetesConfig)
	util.CheckError(err)
}

func getLoadBalancerIP(clienSet kubernetes.Interface, kubenetesNamespace string, maxRetries int) (string, error) {
	var ip string

	for retries := 0; retries < maxRetries; retries++ {
		loadbalancer, err := clienSet.ExtensionsV1beta1().Ingresses(kubenetesNamespace).Get("loadbalancer")

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

func getNamespace(branchName string) string {
	kubenetesNamespace := strings.ToLower(branchName)

	if kubenetesNamespace == "" || kubenetesNamespace == stagingEnvironment || kubenetesNamespace == "master" {
		kubenetesNamespace = stagingEnvironment
	}

	return kubenetesNamespace
}

func getResourceRecordSets(domain string, cnames []string, ip string) []*dns.ResourceRecordSet {
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

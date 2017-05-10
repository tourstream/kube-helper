package app

import (
	"encoding/base64"
	"log"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
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

func createDNSService() *dns.Service {
	ctx := context.Background()

	client, err := google.DefaultClient(ctx, dns.CloudPlatformScope)
	util.CheckError(err)
	dnsService, err := dns.New(client)
	util.CheckError(err)

	return dnsService
}

func createComputeService() *compute.Service {
	ctx := context.Background()

	client, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	util.CheckError(err)
	computeService, err := compute.New(client)
	util.CheckError(err)

	return computeService
}

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

	clientset, err = kubernetes.NewForConfig(kubernetesConfig)
	util.CheckError(err)
}

func getLoadBalancerIP(kubenetesNamespace string, maxRetries int) string {
	var ip string

	for retries := 0; retries < maxRetries; retries++ {
		loadbalancer, err := clientset.Ingresses(kubenetesNamespace).Get("loadbalancer")

		util.CheckError(err)

		if len(loadbalancer.Status.LoadBalancer.Ingress) > 0 {
			ip = loadbalancer.Status.LoadBalancer.Ingress[0].IP
			break
		}
		log.Print("Waiting for Loadbalancer IP")
		time.Sleep(time.Second * 5)
	}
	if ip == "" {
		log.Print("No Loadbalancer IP found")
		return ""
	}
	log.Printf("Loadbalancer IP : %s", ip)

	return ip
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
		&dns.ResourceRecordSet{
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

func waitForStaticIPToBeDeleted(projectID string, addressName string, maxRetries int) {
	computeService := createComputeService()

	addressList, err := computeService.GlobalAddresses.List(projectID).Do()
	util.CheckError(err)
	for _, address := range addressList.Items {
		if (address.Name == addressName) {
			for retries := 0; retries < maxRetries; retries++ {
				_, err := computeService.GlobalAddresses.Get(projectID, address.Name).Do()
				if (err != nil) {
					break;
				}
				log.Printf("Waiting for IP \"%s\" to be released", address.Name)
				time.Sleep(time.Second * 5)
			}
		}
	}
}

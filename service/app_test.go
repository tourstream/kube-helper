package service

import (
	"testing"
	"kube-helper/loader"
	"k8s.io/client-go/kubernetes/fake"
	"github.com/stretchr/testify/assert"
	testing_k8s "k8s.io/client-go/testing"
	"fmt"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	util_clock "k8s.io/apimachinery/pkg/util/clock"
	"gopkg.in/h2non/gock.v1"
	"time"
)

func TestApplicationService_HasNamespace(t *testing.T) {

	var dataProvider = []struct {
		reaction testing_k8s.ReactionFunc
		expected bool
	}{
		{nilReturnFunc, true},
		{errorReturnFunc, false},
	}
	for _, entry := range dataProvider {

		appService, fakeClientSet := getApplicationService(t, "foobar", loader.Config{})

		fakeClientSet.PrependReactor("get", "namespaces", entry.reaction)

		assert.Equal(t, entry.expected, appService.HasNamespace())
	}
}

func TestApplicationService_GetDomain(t *testing.T) {

	var dataProvider = []struct {
		config    loader.DNSConfig
		namespace string
		expected  string
	}{
		{loader.DNSConfig{DomainSuffix: "-testing"}, "foobar", "foobar-testing"},
		{loader.DNSConfig{BaseDomain: "testing", DomainSpacer: "."}, "foobar", "foobar.testing"},
		{loader.DNSConfig{BaseDomain: "testing"}, "production", "testing"},
	}
	for _, entry := range dataProvider {

		appService, _ := getApplicationService(t, entry.namespace, loader.Config{})

		assert.Equal(t, entry.expected, appService.GetDomain(entry.config), fmt.Sprintf("Test failed for namespace %s", entry.namespace))
	}
}

func TestApplicationService_getGcpLoadBalancerIPWithError(t *testing.T) {

	appService, fakeClientSet := getApplicationService(t, "foobar", loader.Config{})

	fakeClientSet.PrependReactor("list", "ingresses", errorReturnFunc)
	assert.EqualError(t, appService.DeleteByNamespace(), "explode")
}

func TestApplicationService_DeleteByNamespaceWithValidLoadBalancerIp(t *testing.T) {

	defer gock.Off()

	response := `
{
  "kind": "dns#change",
  "status": "done"

}`

	responseAddressList := `
{
  "kind": "compute#addressList",
  "items": [
    {
	  "kind": "compute#address",
	  "name": "foobar-ip"
	}
  ]
}`

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"deletions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		Reply(200).
		JSON(response)

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(responseAddressList)

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(`{"kind": "compute#address","name": "foobar-ip"}`)

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(404)

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
		DNS: loader.DNSConfig{
			ProjectID:    "foobar-dns",
			ManagedZone:  "zone-test",
			DomainSpacer: "-",
			BaseDomain:   "testing",
			CNameSuffix:  []string{"-cname.domain."},
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gcp", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))
	fakeClientSet.PrependReactor("delete-collection", "ingresses", nilReturnFunc)
	fakeClientSet.PrependReactor("delete", "namespaces", nilReturnFunc)

	oldClock := clock
	clock = util_clock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

	defer func() {
		clock = oldClock
	}()

	output := captureOutput(func() {
		assert.NoError(t, appService.DeleteByNamespace())
	})

	assert.Contains(t, output, "Namespace \"foobar\" was deleted\n")
	assert.Contains(t, output, "Loadbalancer IP : 127.0.0.1")
	assert.Contains(t, output, "Deleted DNS Entries for 127.0.0.1")
	assert.Contains(t, output, "Waiting for IP \"foobar-ip\" to be released")
	assert.Contains(t, output, "foobar-ip is deleted and so the ingres with name \"Foobar-Ingress\" is removed")
}

func TestApplicationService_DeleteByNamespace(t *testing.T) {

	appService, fakeClientSet := getApplicationService(t, "foobar", loader.Config{})

	fakeClientSet.PrependReactor("delete", "namespaces", nilReturnFunc)
	output := captureOutput(func() {
		assert.NoError(t, appService.DeleteByNamespace())
	})

	assert.Equal(t, "Namespace \"foobar\" was deleted\n", output)
}

func getApplicationService(t *testing.T, namespace string, config loader.Config) (ApplicationServiceInterface, *fake.Clientset) {
	fakeClientSet := fake.NewSimpleClientset()

	builder := new(Builder)

	dnsService, err := builder.GetDNSService()

	assert.NoError(t, err)

	computeService, err := builder.getComputeService()

	assert.NoError(t, err)

	serviceManagementService, err := builder.getServiceManagementService()

	assert.NoError(t, err)

	return NewApplicationService(fakeClientSet, namespace, config, dnsService, computeService, serviceManagementService), fakeClientSet
}

func createAuthCall() {
	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Persist().
		Reply(200).
		JSON(map[string]string{"foo": "bar"})
}

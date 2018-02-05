package service

import (
	"errors"
	"fmt"
	"kube-helper/loader"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util_clock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes/fake"
	testing_k8s "k8s.io/client-go/testing"
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

func TestApplicationService_HasNamespaceWithPrefix(t *testing.T) {

	validate := func(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

		assert.Equal(t, reflect.Indirect(reflect.ValueOf(action)).FieldByName("Name").String(), "dummy-foobar")
		return true, nil, errors.New("explode")
	}

	var dataProvider = []struct {
		reaction testing_k8s.ReactionFunc
		expected bool
	}{
		{nilReturnFunc, true},
		{validate, false},
	}
	for _, entry := range dataProvider {

		appService, fakeClientSet := getApplicationService(t, "foobar", loader.Config{Namespace: loader.Namespace{Prefix: "dummy"}})

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
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
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

func TestApplicationService_DeleteByNamespaceWithValidLoadBalancerIpAndErrorForDeleteDNS(t *testing.T) {

	defer gock.Off()

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
		ReplyError(errors.New("explode"))

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
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
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
		assert.EqualError(t, appService.DeleteByNamespace(), "Post https://www.googleapis.com/dns/v1/projects/foobar-dns/managedZones/zone-test/changes?alt=json: explode")
	})

	assert.Contains(t, output, "Namespace \"foobar\" was deleted\n")
	assert.Contains(t, output, "Loadbalancer IP : 127.0.0.1")
	assert.Contains(t, output, "Waiting for IP \"foobar-ip\" to be released")
	assert.Contains(t, output, "foobar-ip is deleted and so the ingres with name \"Foobar-Ingress\" is removed")
}

func TestApplicationService_DeleteByNamespaceWithValidLoadBalancerIpAndErrorForDeletion(t *testing.T) {

	defer gock.Off()

	response := `
{
  "kind": "dns#change",
  "status": "done"

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
		ReplyError(errors.New("explode"))

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
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
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

	oldClock := clock
	clock = util_clock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

	defer func() {
		clock = oldClock
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.DeleteByNamespace(), "Get https://www.googleapis.com/compute/v1/projects/testing/global/addresses?alt=json: explode")
	})

	assert.Contains(t, output, "Loadbalancer IP : 127.0.0.1")
}

func TestApplicationService_DeleteByNamespaceWithValidLoadBalancerIpAndErrorForDeletionOfIngress(t *testing.T) {

	defer gock.Off()

	response := `
{
  "kind": "dns#change",
  "status": "done"

}`

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"deletions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		Reply(200).
		JSON(response)

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
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))
	fakeClientSet.PrependReactor("delete-collection", "ingresses", errorReturnFunc)

	oldClock := clock
	clock = util_clock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

	defer func() {
		clock = oldClock
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.DeleteByNamespace(), "explode")
	})

	assert.Contains(t, output, "Loadbalancer IP : 127.0.0.1")
}

func TestApplicationService_DeleteByNamespace(t *testing.T) {

	appService, fakeClientSet := getApplicationService(t, "foobar", loader.Config{})

	fakeClientSet.PrependReactor("delete", "namespaces", nilReturnFunc)
	output := captureOutput(func() {
		assert.NoError(t, appService.DeleteByNamespace())
	})

	assert.Equal(t, "Namespace \"foobar\" was deleted\n", output)
}

func TestApplicationService_DeleteByNamespaceWithError(t *testing.T) {

	appService, fakeClientSet := getApplicationService(t, "foobar", loader.Config{})

	fakeClientSet.PrependReactor("delete", "namespaces", errorReturnFunc)
	assert.EqualError(t, appService.DeleteByNamespace(), "explode")
}

func TestApplicationService_ApplyWithInvalidNamespace(t *testing.T) {

	config := loader.Config{}
	appService, _ := getApplicationService(t, "foo_bar", config)

	assert.EqualError(t, appService.Apply(), "[a-z0-9]([-a-z0-9]*[a-z0-9])? (e.g. '123-abc', regex used for validation is 'my-name')")
}

func TestApplicationService_ApplyWithErrorDuringNamespaceCreation(t *testing.T) {

	config := loader.Config{}
	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	fakeClientSet.PrependReactor("create", "namespaces", errorReturnFunc)

	assert.EqualError(t, appService.Apply(), "explode")
}

func TestApplicationService_ApplyWithEndpointsWithError(t *testing.T) {

	config := loader.Config{
		Endpoints: loader.Endpoints{
			Enabled: true,
		},
		DNS: loader.DNSConfig{
			BaseDomain:   "dummy.local.",
			DomainSpacer: "-",
		},
	}
	appService, _ := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	createAuthCall()

	gock.New("https://servicemanagement.googleapis.com").
		Get("/v1/services/foobar-dummy.local/configs").
		ReplyError(errors.New("explode"))

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "Get https://servicemanagement.googleapis.com/v1/services/foobar-dummy.local/configs?alt=json: explode")
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
	assert.Empty(t, os.Getenv("ENDPOINT_VERSION"))
	assert.Empty(t, os.Getenv("ENDPOINT_DOMAIN"))
	assert.True(t, gock.IsDone())
}

func TestApplicationService_ApplyWithEndpoints(t *testing.T) {

	config := loader.Config{
		Endpoints: loader.Endpoints{
			Enabled: true,
		},
		DNS: loader.DNSConfig{
			BaseDomain:   "dummy.local.",
			DomainSpacer: "-",
		},
	}
	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	response := `
{
  "serviceConfigs": [
    {
      "name": "foobar-dummy.local",
      "title": "foobar API",
      "documentation": {},
      "usage": {},
      "id": "2017-08-30r0"
    },
    {
      "name": "foobar-dummy.local",
      "title": "foobar API",
      "documentation": {
        "summary": "foobar API"
      },
      "usage": {},
      "id": "2017-08-29r0"
    }
  ]
}

	`
	createAuthCall()

	gock.New("https://servicemanagement.googleapis.com").
		Get("/v1/services/foobar-dummy.local/configs").
		Reply(200).
		JSON(response)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
		gock.Off()
	}()

	output := captureOutput(func() {
		assert.NoError(t, appService.Apply())
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
	assert.Contains(t, output, "There are 0 pods in the cluster\n")
	assert.Equal(t, "2017-08-30r0", os.Getenv("ENDPOINT_VERSION"))
	assert.Equal(t, "foobar-dummy.local", os.Getenv("ENDPOINT_DOMAIN"))
	assert.True(t, gock.IsDone())
}

func TestApplicationService_ApplyWithErrorForGetPods(t *testing.T) {

	config := loader.Config{}
	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	fakeClientSet.PrependReactor("list", "pods", errorReturnFunc)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "explode")
	})

	assert.Equal(t, output, "Namespace \"foobar\" was generated\n")
}

func TestApplicationService_ApplyWithErrorInReplace(t *testing.T) {

	config := loader.Config{}
	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return errors.New("explode")
	}

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "explode")
	})

	assert.Equal(t, output, "Namespace \"foobar\" was generated\n")
}

func TestApplicationService_Apply(t *testing.T) {

	config := loader.Config{}
	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
	}()

	output := captureOutput(func() {
		assert.NoError(t, appService.Apply())
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
	assert.Contains(t, output, "There are 0 pods in the cluster\n")
}

func TestApplicationService_ApplyWithErrorForImageService(t *testing.T) {

	config := loader.Config{}
	appService, _ := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	serviceBuilderMock.On("GetImagesService").Return(nil, errors.New("explode"))

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "explode")
	})

	assert.Equal(t, output, "Namespace \"foobar\" was generated\n")
}

func TestApplicationService_ApplyWithDNSAndErrorForLoadBalancerIp(t *testing.T) {

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

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	fakeClientSet.PrependReactor("list", "ingresses", errorReturnFunc)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "explode")
	})

	assert.Equal(t, output, "Namespace \"foobar\" was generated\n")
}

func TestApplicationService_ApplyWithDNSAndErrorForWaitungOnLoadbalancerIp(t *testing.T) {

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

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))
	fakeClientSet.PrependReactor("get", "ingresses", errorReturnFunc)

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
		gock.Off()
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "explode")
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
}

func TestApplicationService_ApplyWithDNSAndErrorForWaitungOnLoadbalancerIpWithGet(t *testing.T) {

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

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	singleObject := &v1beta1.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{IP: ""},
				},
			}}}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))
	fakeClientSet.PrependReactor("get", "ingresses", getObjectReturnFunc(singleObject))

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
		gock.Off()
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "no Loadbalancer IP found")
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
}

func TestApplicationService_ApplyWithDNSAndErrorForWaitungOnLoadbalancerIpWithRetry(t *testing.T) {

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

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	singleObject := &v1beta1.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{}}}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))
	fakeClientSet.PrependReactor("get", "ingresses", getObjectReturnFunc(singleObject))

	oldClock := clock
	clock = util_clock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
		gock.Off()
		clock = oldClock

	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "no Loadbalancer IP found")
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
	assert.Contains(t, output, "Waiting for Loadbalancer IP\n")
}

func TestApplicationService_ApplyWithDNSAndErrorForDomainCreation(t *testing.T) {

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

	createAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
		gock.Off()
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "Post https://www.googleapis.com/dns/v1/projects/foobar-dns/managedZones/zone-test/changes?alt=json: explode")
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
	assert.Contains(t, output, "Loadbalancer IP : 127.0.0.1\n")
}

func TestApplicationService_ApplyWithDNS(t *testing.T) {

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
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		Reply(200).
		JSON(response)

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(responseAddressList)

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(`{"kind": "compute#address","name": "foobar-ip"}`)

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(404)

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	oldServiceBuilder := serviceBuilder
	serviceBuilderMock := new(MockBuilderInterface)

	serviceBuilder = serviceBuilderMock

	imagesMock := new(MockImagesInterface)
	kindMock := new(MockKindInterface)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	serviceBuilderMock.On("GetImagesService").Return(imagesMock, nil)
	serviceBuilderMock.On("GetKindService", fakeClientSet, imagesMock, config).Return(kindMock)

	serviceBuilder = serviceBuilderMock

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
		serviceBuilder = oldServiceBuilder
		gock.Off()
	}()

	output := captureOutput(func() {
		assert.NoError(t, appService.Apply())
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
	assert.Contains(t, output, "Loadbalancer IP : 127.0.0.1\n")
	assert.Contains(t, output, "Created DNS Entries for 127.0.0.1\n")
	assert.Contains(t, output, "There are 0 pods in the cluster\n")
}

func getApplicationService(t *testing.T, namespace string, config loader.Config) (ApplicationServiceInterface, *fake.Clientset) {
	fakeClientSet := fake.NewSimpleClientset()

	builder := new(Builder)

	dnsService, err := builder.GetDNSService()

	assert.NoError(t, err)

	computeService, err := builder.GetComputeService()

	assert.NoError(t, err)

	serviceManagementService, err := builder.getServiceManagementService()

	assert.NoError(t, err)

	return NewApplicationService(fakeClientSet, namespace, config, dnsService, computeService, serviceManagementService), fakeClientSet
}

func createAuthCall() {
	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"access_token": "bar"})
}

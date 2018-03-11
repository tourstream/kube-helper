package app

import (
	"errors"
	"fmt"
	"kube-helper/loader"
	"kube-helper/mocks"
	"os"
	"reflect"
	"testing"
	"time"

	"bytes"

	testingKube "kube-helper/testing"

	"kube-helper/service/image"
	"kube-helper/service/kind"

	"kube-helper/service/builder"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilClock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	testingK8s "k8s.io/client-go/testing"
)

func TestApplicationService_HasNamespace(t *testing.T) {

	var dataProvider = []struct {
		reaction testingK8s.ReactionFunc
		expected bool
	}{
		{testingKube.NilReturnFunc, true},
		{testingKube.ErrorReturnFunc, false},
	}

	oldServiceBuilder := serviceBuilder

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	for _, entry := range dataProvider {

		serviceBuilderMock, fakeClientSet := getBuilderMock(t, loader.Config{}, nil)

		serviceBuilder = serviceBuilderMock

		appService, err := NewApplicationService("foobar", loader.Config{})

		assert.NoError(t, err)

		fakeClientSet.PrependReactor("get", "namespaces", entry.reaction)

		assert.Equal(t, entry.expected, appService.HasNamespace())
	}
}

func TestApplicationService_HasNamespaceWithPrefix(t *testing.T) {

	validate := func(action testingK8s.Action) (handled bool, ret runtime.Object, err error) {

		assert.Equal(t, reflect.Indirect(reflect.ValueOf(action)).FieldByName("Name").String(), "dummy-foobar")
		return true, nil, errors.New("explode")
	}

	var dataProvider = []struct {
		reaction testingK8s.ReactionFunc
		expected bool
	}{
		{testingKube.NilReturnFunc, true},
		{validate, false},
	}

	oldServiceBuilder := serviceBuilder
	config := loader.Config{Namespace: loader.Namespace{Prefix: "dummy"}}

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	for _, entry := range dataProvider {

		serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

		serviceBuilder = serviceBuilderMock

		appService, err := NewApplicationService("foobar", config)

		assert.NoError(t, err)

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

	oldServiceBuilder := serviceBuilder
	config := loader.Config{}

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	for _, entry := range dataProvider {

		serviceBuilderMock, _ := getBuilderMock(t, config, nil)

		serviceBuilder = serviceBuilderMock

		appService, err := NewApplicationService(entry.namespace, config)

		assert.NoError(t, err)

		assert.Equal(t, entry.expected, appService.GetDomain(entry.config), fmt.Sprintf("Test failed for namespace %s", entry.namespace))
	}
}

func TestApplicationService_getGcpLoadBalancerIPWithError(t *testing.T) {

	oldServiceBuilder := serviceBuilder
	config := loader.Config{}

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.ErrorReturnFunc)
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

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"deletions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		Reply(200).
		JSON(response)

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(responseAddressList)

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(`{"kind": "compute#address","name": "foobar-ip"}`)

	testingKube.CreateAuthCall()

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

	oldServiceBuilder := serviceBuilder

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))
	fakeClientSet.PrependReactor("delete-collection", "ingresses", testingKube.NilReturnFunc)
	fakeClientSet.PrependReactor("delete", "namespaces", testingKube.NilReturnFunc)

	oldClock := clock
	clock = utilClock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

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

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"deletions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(responseAddressList)

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(`{"kind": "compute#address","name": "foobar-ip"}`)

	testingKube.CreateAuthCall()

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

	oldServiceBuilder := serviceBuilder

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))
	fakeClientSet.PrependReactor("delete-collection", "ingresses", testingKube.NilReturnFunc)
	fakeClientSet.PrependReactor("delete", "namespaces", testingKube.NilReturnFunc)

	oldClock := clock
	clock = utilClock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

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

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"deletions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		Reply(200).
		JSON(response)

	testingKube.CreateAuthCall()

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

	oldServiceBuilder := serviceBuilder

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))
	fakeClientSet.PrependReactor("delete-collection", "ingresses", testingKube.NilReturnFunc)

	oldClock := clock
	clock = utilClock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

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

	testingKube.CreateAuthCall()

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

	oldServiceBuilder := serviceBuilder

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))
	fakeClientSet.PrependReactor("delete-collection", "ingresses", testingKube.ErrorReturnFunc)

	oldClock := clock
	clock = utilClock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

	defer func() {
		clock = oldClock
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.DeleteByNamespace(), "explode")
	})

	assert.Contains(t, output, "Loadbalancer IP : 127.0.0.1")
}

func TestApplicationService_DeleteByNamespace(t *testing.T) {

	oldServiceBuilder := serviceBuilder
	config := loader.Config{}

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	fakeClientSet.PrependReactor("delete", "namespaces", testingKube.NilReturnFunc)
	output := captureOutput(func() {
		assert.NoError(t, appService.DeleteByNamespace())
	})

	assert.Equal(t, "Namespace \"foobar\" was deleted\n", output)
}

func TestApplicationService_DeleteByNamespaceWithError(t *testing.T) {

	oldServiceBuilder := serviceBuilder
	config := loader.Config{}

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	fakeClientSet.PrependReactor("delete", "namespaces", testingKube.ErrorReturnFunc)
	assert.EqualError(t, appService.DeleteByNamespace(), "explode")
}

func TestApplicationService_ApplyWithInvalidNamespace(t *testing.T) {

	oldServiceBuilder := serviceBuilder
	config := loader.Config{}

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, _ := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foo_bar", config)

	assert.NoError(t, err)

	assert.EqualError(t, appService.Apply(), "[a-z0-9]([-a-z0-9]*[a-z0-9])? (e.g. '123-abc', regex used for validation is 'my-name')")
}

func TestApplicationService_ApplyWithErrorDuringNamespaceCreation(t *testing.T) {

	oldServiceBuilder := serviceBuilder
	config := loader.Config{}

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	fakeClientSet.PrependReactor("create", "namespaces", testingKube.ErrorReturnFunc)

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
	oldServiceBuilder := serviceBuilder

	defer func() {
		serviceBuilder = oldServiceBuilder
	}()

	serviceBuilderMock, _ := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	testingKube.CreateAuthCall()

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

	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

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
	testingKube.CreateAuthCall()

	gock.New("https://servicemanagement.googleapis.com").
		Get("/v1/services/foobar-dummy.local/configs").
		Reply(200).
		JSON(response)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
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
	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	fakeClientSet.PrependReactor("list", "pods", testingKube.ErrorReturnFunc)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "explode")
	})

	assert.Equal(t, output, "Namespace \"foobar\" was generated\nNo suitable ingress found\nNo Annotations to process")
}

func TestApplicationService_ApplyWithErrorInReplace(t *testing.T) {

	config := loader.Config{}
	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return errors.New("explode")
	}

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
	}()

	output := captureOutput(func() {
		assert.EqualError(t, appService.Apply(), "explode")
	})

	assert.Equal(t, output, "Namespace \"foobar\" was generated\n")
}

func TestApplicationService_Apply(t *testing.T) {

	config := loader.Config{}
	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
	}()

	output := captureOutput(func() {
		assert.NoError(t, appService.Apply())
	})

	assert.Contains(t, output, "Namespace \"foobar\" was generated\n")
	assert.Contains(t, output, "There are 0 pods in the cluster\n")
}

func TestApplicationService_ApplyWithErrorForImageService(t *testing.T) {

	config := loader.Config{}
	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	serviceBuilderMock, _ := getBuilderMock(t, config, nil)

	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
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

	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.ErrorReturnFunc)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
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

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))
	fakeClientSet.PrependReactor("get", "ingresses", testingKube.ErrorReturnFunc)

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
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

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	singleObject := &v1beta1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{IP: ""},
				},
			}}}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))
	fakeClientSet.PrependReactor("get", "ingresses", testingKube.GetObjectReturnFunc(singleObject))

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
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

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	singleObject := &v1beta1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{}}}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))
	fakeClientSet.PrependReactor("get", "ingresses", testingKube.GetObjectReturnFunc(singleObject))

	oldClock := clock
	clock = utilClock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
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

	testingKube.CreateAuthCall()

	gock.New("https://www.googleapis.com").
		Post("/dns/v1/projects/foobar-dns/managedZones/zone-test/changes").
		MatchType("json").
		BodyString(`{"additions":[{"name":"foobar-testing","rrdatas":["127.0.0.1"],"ttl":300,"type":"A"},{"name":"foobar-cname.domain.","rrdatas":["foobar-testing"],"ttl":300,"type":"CNAME"}]}`).
		ReplyError(errors.New("explode"))

	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
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

	testingKube.CreateAuthCall()

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

	oldServiceBuilder := serviceBuilder
	oldKindServiceCreator := kindServiceCreator

	defer func() {
		serviceBuilder = oldServiceBuilder
		kindServiceCreator = oldKindServiceCreator
	}()

	imagesMock := new(mocks.ImagesInterface)
	kindMock := new(mocks.KindInterface)
	serviceBuilderMock, fakeClientSet := getBuilderMock(t, config, imagesMock)

	kindServiceCreator = mockkindServiceCreator(t, fakeClientSet, imagesMock, config, kindMock)
	serviceBuilder = serviceBuilderMock

	appService, err := NewApplicationService("foobar", config)

	assert.NoError(t, err)

	oldLReplaceFunc := replaceVariablesInFile

	replaceVariablesInFile = func(fileSystem afero.Fs, path string, functionCall loader.Callable) error {
		return functionCall([]string{})
	}

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: metaV1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"kubernetes.io/ingress.class": "gce", "ingress.kubernetes.io/static-ip": "foobar-ip"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "127.0.0.1"},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", testingKube.GetObjectReturnFunc(list))

	kindMock.On("ApplyKind", "foobar", []string{}, "foobar").Return(nil)
	kindMock.On("CleanupKind", "foobar").Return(nil)

	defer func() {
		replaceVariablesInFile = oldLReplaceFunc
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

func getBuilderMock(t *testing.T, config loader.Config, imageMock image.ImagesInterface) (builder.ServiceBuilderInterface, *fake.Clientset) {
	builderService := new(builder.Builder)

	dnsService, err := builderService.GetDNSService()

	assert.NoError(t, err)

	computeService, err := builderService.GetComputeService()

	assert.NoError(t, err)

	serviceManagementService, err := builderService.GetServiceManagementService()

	assert.NoError(t, err)

	fakeClientSet := fake.NewSimpleClientset()
	serviceBuilderMock := new(mocks.ServiceBuilderInterface)

	var mockedImageError error

	if imageMock == nil {
		mockedImageError = errors.New("explode")
	}

	serviceBuilderMock.On("GetClientSet", config).Return(fakeClientSet, nil)
	serviceBuilderMock.On("GetServiceManagementService").Return(serviceManagementService, nil)
	serviceBuilderMock.On("GetDNSService").Return(dnsService, nil)
	serviceBuilderMock.On("GetComputeService").Return(computeService, nil)
	serviceBuilderMock.On("GetImagesService").Return(imageMock, mockedImageError)

	return serviceBuilderMock, fakeClientSet
}

func captureOutput(f func()) string {
	oldWriter := writer
	var buf bytes.Buffer
	defer func() {
		writer = oldWriter
	}()
	writer = &buf
	f()
	return buf.String()
}

func mockkindServiceCreator(t *testing.T, expectedClientSet kubernetes.Interface, expectedImagesService image.ImagesInterface, expectedConfig loader.Config, serviceMock kind.KindInterface) func(client kubernetes.Interface, imagesService image.ImagesInterface, config loader.Config) kind.KindInterface {
	return func(client kubernetes.Interface, imagesService image.ImagesInterface, config loader.Config) kind.KindInterface {
		assert.Equal(t, expectedConfig, config)
		assert.Equal(t, expectedClientSet, client)
		assert.Equal(t, expectedImagesService, imagesService)

		return serviceMock
	}
}

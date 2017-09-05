package service

import (
	"testing"
	"kube-helper/loader"
	"k8s.io/client-go/kubernetes/fake"
	"github.com/stretchr/testify/assert"
	testing_k8s "k8s.io/client-go/testing"
	"fmt"
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

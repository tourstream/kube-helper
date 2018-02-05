package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"kube-helper/loader"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gopkg.in/h2non/gock.v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/api/core/v1"
	"errors"
	"time"
	util_clock "k8s.io/apimachinery/pkg/util/clock"
)

func TestAnnotations_HandleIngressAnnotationOnApply(t *testing.T) {
	defer gock.Off()
	createAuthCall()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip1,,test-ip:8080"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		assert.NoError(t, appService.HandleIngressAnnotationOnApply())
	})
	assert.Contains(t, output, "Invalid Address \"test-ip1\"\nEmpty Address\nInvalid Address \"test-ip:8080\"")
}

func TestAnnotations_HandleIngressAnnotationOnApply_preSharedCert(t *testing.T) {
	defer gock.Off()
	createAuthCall()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/pre-shared-cert": ""}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()

	assert.Contains(t, err.Error(), "No Certificate to add")
}

func TestAnnotations_HandleIngressAnnotationOnApply_notSupportedAnnotation(t *testing.T) {
	defer gock.Off()
	createAuthCall()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/foo": "test-ip1,,test-ip:8080"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Contains(t, err.Error(), "not supported Annotation")
}

func TestAnnotations_HandleIngressAnnotationOnApply_ReadingConfigAddresses_handlingEmptyConfig(t *testing.T) {
	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{{}},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		err := appService.HandleIngressAnnotationOnApply()
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "No Annotations to process")
}

func TestAnnotations_HandleIngressAnnotationOnApply_ReadingConfigAddresses_handlesConfigEmpty(t *testing.T) {
	config := loader.Config{}
	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	fakeClientSet.PrependReactor("list", "ingresses", errorReturnFunc)

	err := appService.HandleIngressAnnotationOnApply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "explode")
}

func TestAnnotations_AddNewRulesToLoadBalancer(t *testing.T) {
	defer gock.Off()

	projectId = "testing"
	kb8Namespace = "foobar"

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules/test-ip1-fr-80").
		Reply(404).SetError(errors.New("Error 404"))

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules/test-ip1-fr-443").
		Reply(404).SetError(errors.New("Error 404"))

	var newRule = []byte(`{
	"IPAddress":"127.0.0.1",
	"IPProtocol":"TCP",
	"loadBalancingScheme":"EXTERNAL",
	"name":"test-ip1-fr-80",
	"portRange":"80-80",
	"target":"https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpProxies/k8s-um-portal-production-loadb-target-proxy-4"}`)

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Post("/compute/v1/projects/testing/global/forwardingRules").
		MatchType("json").
		JSON(newRule). //matches body
		Reply(200).
		JSON(map[string]string{})

	var newRule2 = []byte(`{
	"IPAddress":"127.0.0.1",
	"IPProtocol":"TCP",
	"loadBalancingScheme":"EXTERNAL",
	"name":"test-ip1-fr-443",
	"portRange":"443-443",
	"target":"https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpsProxies/k8s-um-portal-production-loadb-target-proxy-4"}`)

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Post("/compute/v1/projects/testing/global/forwardingRules").
		MatchType("json").
		JSON(newRule2). //matches body
		Reply(200).
		JSON(map[string]string{})

	mockGetForwardingRules()
	mockGetForwardingRules()
	mockGetAddresses()
	mockGetAddresses()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip1:80,test-ip1:443"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		assert.NoError(t, appService.HandleIngressAnnotationOnApply())
	})

	assert.Contains(t, output, "Adress {test-ip1 80} added\nAdress {test-ip1 443} added")
}
func TestAnnotations_AddNewRulesToLoadBalancer_FailCreateNewRule(t *testing.T) {
	defer gock.Off()

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules").
		Reply(502).SetError(errors.New("failed"))

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip1:80,test-ip1:443"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		err := appService.HandleIngressAnnotationOnApply()
		assert.Contains(t, err.Error(), "Get https://www.googleapis.com/compute/v1/projects/testing/global/forwardingRules?alt=json: failed")
	})

	assert.Contains(t, output, "Get https://www.googleapis.com/compute/v1/projects/testing/global/forwardingRules?alt=json: failed")
}

func TestAnnotations_AddNewRulesToLoadBalancer_failInsertRule(t *testing.T) {
	defer gock.Off()

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules/test-ip1-fr-80").
		Reply(404).SetError(errors.New("Error 404"))

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules/test-ip1-fr-443").
		Reply(404).SetError(errors.New("Error 404"))

	var newRule = []byte(`{
	"IPAddress":"127.0.0.1",
	"IPProtocol":"TCP",
	"loadBalancingScheme":"EXTERNAL",
	"name":"test-ip1-fr-80",
	"portRange":"80-80",
	"target":"https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpProxies/k8s-um-portal-production-loadb-target-proxy-4"}`)

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Post("/compute/v1/projects/testing/global/forwardingRules").
		MatchType("json").
		JSON(newRule). //matches body
		Reply(200).
		JSON(map[string]string{})

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Post("/compute/v1/projects/testing/global/forwardingRules").
		Reply(502).SetError(errors.New("failed"))

	mockGetForwardingRules()
	mockGetForwardingRules()
	mockGetAddresses()
	mockGetAddresses()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip1:80,test-ip1:443"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		err := appService.HandleIngressAnnotationOnApply()
		assert.Contains(t, err.Error(), "Post https://www.googleapis.com/compute/v1/projects/testing/global/forwardingRules?alt=json: failed")
	})

	assert.Contains(t, output, "Adress {test-ip1 80} added")
}

func TestAnnotations_CreateNewRule_FailGetAddresses(t *testing.T) {
	defer gock.Off()

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules/test-ip1-fr-80").
		Reply(404).SetError(errors.New("Error 404"))

	var newRule = []byte(`{
	"IPAddress":"127.0.0.1",
	"IPProtocol":"TCP",
	"loadBalancingScheme":"EXTERNAL",
	"name":"test-ip1-fr-80",
	"portRange":"80-80",
	"target":"https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpProxies/k8s-um-portal-production-loadb-target-proxy-4"}`)

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Post("/compute/v1/projects/testing/global/forwardingRules").
		MatchType("json").
		JSON(newRule). //matches body
		Reply(200).
		JSON(map[string]string{})

	mockGetForwardingRules()

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(502).SetError(errors.New("failed"))

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip1:80"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		err := appService.HandleIngressAnnotationOnApply()
		assert.Contains(t, err.Error(), "Get https://www.googleapis.com/compute/v1/projects/testing/global/addresses?alt=json: failed")
	})

	assert.Contains(t, output, "Get https://www.googleapis.com/compute/v1/projects/testing/global/addresses?alt=json: failed")
}

func TestAnnotations_CreateNewRule_AddressesInUse(t *testing.T) {
	defer gock.Off()

	mockGetForwardingRules()
	mockGetAddresses()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip3:80"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Contains(t, err.Error(), "Ip Address already used")
}

func TestAnnotations_CreateNewRule_RuleExists(t *testing.T) {
	defer gock.Off()

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules/test-ip1-fr-80").
		Reply(200).
		JSON(`{
		"IPAddress": "127.0.0.1",
		"IPProtocol": "TCP",
		"creationTimestamp": "2018-02-12T00:59:20.975-08:00",
		"description": "",
		"id": "1",
		"kind": "compute#forwardingRule",
		"loadBalancingScheme": "EXTERNAL",
		"name": "fti-fr-ipv4-https-foobar",
		"portRange": "443-443",
		"selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/forwardingRules/fti-fr-ipv4-https",
		"target": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpsProxies/k8s-um-portal-production-loadb-target-proxy-4"
	}`)

	mockGetForwardingRules()
	mockGetAddresses()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip1:80"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Contains(t, err.Error(), "Rule test-ip1-fr-80 already exists")
}

func TestAnnotations_CreateNewRule_RuleLoadError(t *testing.T) {
	defer gock.Off()

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules/test-ip1-fr-80").
		Reply(502).SetError(errors.New("failed"))

	mockGetForwardingRules()
	mockGetAddresses()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip1:80"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Contains(t, err.Error(), "Get https://www.googleapis.com/compute/v1/projects/testing/global/forwardingRules/test-ip1-fr-80?alt=json: failed")
}

func TestAnnotations_GetRuleToCopy_WaitAndNoMatch(t *testing.T) {
	oldClock := clock
	clock = util_clock.NewFakeClock(time.Date(2014, 1, 1, 3, 0, 30, 0, time.UTC))
	defer func() {
		clock = oldClock
	}()

	defer gock.Off()
	createAuthCall()

	for i := 0; i < 60; i++ {
		mockGetForwardingRules()
	}

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "barfoo", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip6:80"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		err := appService.HandleIngressAnnotationOnApply()
		assert.Contains(t, err.Error(), "No existing rule for namespace found")
	})

	assert.Contains(t, output, "Waiting for first Rule to be appended\nWaiting for first Rule to be appended\nWaiting for first Rule to be appended")
}

func TestAnnotations_GetIpAddressByName_NoMatch(t *testing.T) {
	defer gock.Off()
	createAuthCall()
	mockGetForwardingRules()
	mockGetAddresses()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/named-ip-address": "test-ip6:80"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Contains(t, err.Error(), "Ip Address not found")
}

func TestAnnotations_AddCertificatesToHttpsProxies(t *testing.T) {
	defer gock.Off()

	certificates := []string{"fti-nl", "fti-fr"}

	for _, certificate := range certificates {
		mockGetCertificate(certificate)
	}

	mockGetHttpsProxies()
	var certificateList = []byte(`{"sslCertificates":["https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-fr","https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/gcloud-fti-group-com","https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-nl"]}`)

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Post("/compute/v1/projects/testing/targetHttpsProxies/k8s-um-portal-foobar-loadb-target-proxy-6/setSslCertificates").
		MatchType("json").
		JSON(certificateList). //matches body
		Reply(200).
		JSON(map[string]string{})

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/pre-shared-cert": "fti-nl,fti-fr"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		err := appService.HandleIngressAnnotationOnApply()
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Certificates [https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-nl https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-fr] added to k8s-um-portal-foobar-loadb-target-proxy-6")
}

func TestAnnotations_AddCertificatesToHttpsProxies_OneEmptyString(t *testing.T) {
	defer gock.Off()

	mockGetCertificate("fti-nl")

	mockGetHttpsProxies()
	var certificateList = []byte(`{"sslCertificates":["https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-fr","https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/gcloud-fti-group-com","https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-nl"]}`)

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Post("/compute/v1/projects/testing/targetHttpsProxies/k8s-um-portal-foobar-loadb-target-proxy-6/setSslCertificates").
		MatchType("json").
		JSON(certificateList). //matches body
		Reply(200).
		JSON(map[string]string{})

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/pre-shared-cert": ",fti-nl,"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	output := captureOutput(func() {
		err := appService.HandleIngressAnnotationOnApply()
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Certificates [https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-nl] added to k8s-um-portal-foobar-loadb-target-proxy-6")
}

func TestAnnotations_AddCertificatesToHttpsProxies_ErrorGetCertificate(t *testing.T) {
	defer gock.Off()

	projectId = "testing"
	kb8Namespace = "foobar"

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/sslCertificates/fti-fr").
		Reply(502).SetError(errors.New("Failed"))


	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/pre-shared-cert": "fti-fr"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Get https://www.googleapis.com/compute/v1/projects/testing/global/sslCertificates/fti-fr?alt=json: Failed")
}

func TestAnnotations_AddCertificatesToHttpsProxies_EmptyCertList(t *testing.T) {
	defer gock.Off()
	createAuthCall()

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/pre-shared-cert": ",,"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No Certificate to add")
}

func TestAnnotations_AddCertificatesToHttpsProxies_ErrorGettingProxies(t *testing.T) {
	defer gock.Off()

	certificates := []string{"fti-nl", "fti-fr"}

	for _, certificate := range certificates {
		mockGetCertificate(certificate)
	}

	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/targetHttpsProxies").
		Reply(502).SetError(errors.New("failed"))


	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/pre-shared-cert": "fti-nl,fti-fr"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Get https://www.googleapis.com/compute/v1/projects/testing/global/targetHttpsProxies?alt=json: failed")
}

func TestAnnotations_AddCertificatesToHttpsProxies_AppendFails(t *testing.T) {
	defer gock.Off()

	certificates := []string{"fti-nl", "fti-fr"}

	for _, certificate := range certificates {
		mockGetCertificate(certificate)
	}

	mockGetHttpsProxies()
	var certificateList = []byte(`{"sslCertificates":["https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-fr","https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/gcloud-fti-group-com","https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-nl"]}`)


	createAuthCall()
	gock.New("https://www.googleapis.com").
		Post("/compute/v1/projects/testing/targetHttpsProxies/k8s-um-portal-foobar-loadb-target-proxy-6/setSslCertificates").
		MatchType("json").
		JSON(certificateList). //matches body
		Reply(502).SetError(errors.New("failed"))

	config := loader.Config{
		Cluster: loader.Cluster{
			Type:      "gcp",
			ProjectID: "testing",
		},
	}

	appService, fakeClientSet := getApplicationService(t, "foobar", config)

	list := &v1beta1.IngressList{
		Items: []v1beta1.Ingress{
			{},
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "Foobar-Ingress", Annotations: map[string]string{"tourstream.eu/pre-shared-cert": "fti-nl,fti-fr"}},
				Status: v1beta1.IngressStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: ""},
						},
					}}},
		},
	}

	fakeClientSet.PrependReactor("list", "ingresses", getObjectReturnFunc(list))

	err := appService.HandleIngressAnnotationOnApply()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Post https://www.googleapis.com/compute/v1/projects/testing/targetHttpsProxies/k8s-um-portal-foobar-loadb-target-proxy-6/setSslCertificates?alt=json: failed")
}

func mockGetAddresses() {
	var addresses = []byte(`{"items":[{
    "address": "127.0.0.1",
    "creationTimestamp": "2017-11-08T05:03:44.487-08:00",
    "description": "",
    "id": "1",
    "kind": "compute#address",
    "name": "test-ip1",
    "selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/addresses/test-ip",
    "status": "UNUSED",
    "users": []
  },{
    "address": "127.0.0.1",
    "creationTimestamp": "2017-11-08T05:03:44.487-08:00",
    "description": "",
    "id": "1",
    "kind": "compute#address",
    "name": "test-ip2",
    "selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/addresses/test-ip",
    "status": "UNUSED",
    "users": []
  },{
    "address": "127.0.0.1",
    "creationTimestamp": "2017-11-08T05:03:44.487-08:00",
    "description": "",
    "id": "1",
    "kind": "compute#address",
    "name": "test-ip3",
    "selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/addresses/test-ip",
    "status": "IN USE",
    "users": []
  }
]}`)


	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/addresses").
		Reply(200).
		JSON(addresses)
}

func mockGetForwardingRules() {
	var forwardingRules = []byte(`{"items":[{
		"IPAddress": "127.0.0.1",
		"IPProtocol": "TCP",
		"creationTimestamp": "2018-02-12T00:59:20.975-08:00",
		"description": "",
		"id": "1",
		"kind": "compute#forwardingRule",
		"loadBalancingScheme": "EXTERNAL",
		"name": "fti-fr-ipv4-https-foobar",
		"portRange": "443-443",
		"selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/forwardingRules/fti-fr-ipv4-https",
		"target": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpsProxies/k8s-um-portal-production-loadb-target-proxy-4"
	},
{
		"IPAddress": "127.0.0.1",
		"IPProtocol": "TCP",
		"creationTimestamp": "2018-02-12T00:59:20.975-08:00",
		"description": "",
		"id": "1",
		"kind": "compute#forwardingRule",
		"loadBalancingScheme": "EXTERNAL",
		"name": "fti-fr-ipv4-http-foobar",
		"portRange": "80-80",
		"selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/forwardingRules/fti-fr-ipv4-https",
		"target": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpProxies/k8s-um-portal-production-loadb-target-proxy-4"
	}]}`)


	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/forwardingRules").
		Reply(200).
		JSON(forwardingRules)
}

func mockGetCertificate(name string) {
	var certi = []byte(`{
	"certificate": "",
    "creationTimestamp": "2017-07-19T01:08:10.312-07:00",
    "id": "5393002595470192005",
    "kind": "compute#sslCertificate",
    "name": "` + name + `",
    "selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/` + name + `"
}`)


	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/sslCertificates/" + name).
		Reply(200).
		JSON(certi)
}

func mockGetHttpsProxies() {
	var httpsProxies = []byte(`{"items":[{
"creationTimestamp": "2018-02-12T01:01:23.856-08:00",
"id": "1914888759999785228",
"kind": "compute#targetHttpsProxy",
"name": "k8s-um-portal-production-loadb-target-proxy-6",
"selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpsProxies/k8s-um-portal-production-loadb-target-proxy-6",
"sslCertificates": [
"https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-fr",
"https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/gcloud-fti-group-com"
],
"urlMap": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/urlMaps/k8s-um-portal-production-loadbalancer--6ce2007c4dccdc3f"
},
{
"creationTimestamp": "2018-02-12T01:01:23.856-08:00",
"id": "1914888759999785228",
"kind": "compute#targetHttpsProxy",
"name": "k8s-um-portal-foobar-loadb-target-proxy-6",
"selfLink": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/targetHttpsProxies/k8s-um-portal-production-loadb-target-proxy-6",
"sslCertificates": [
"https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/fti-fr",
"https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/sslCertificates/gcloud-fti-group-com"
],
"urlMap": "https://www.googleapis.com/compute/v1/projects/e-tourism-suite/global/urlMaps/k8s-um-portal-production-loadbalancer--6ce2007c4dccdc3f"
}
]}`)


	createAuthCall()
	gock.New("https://www.googleapis.com").
		Get("/compute/v1/projects/testing/global/targetHttpsProxies").
		Reply(200).
		JSON(httpsProxies)
}

package service

import (
	"testing"
	"kube-helper/loader"
	"k8s.io/client-go/kubernetes/fake"
	testing_k8s "k8s.io/client-go/testing"
	"k8s.io/apimachinery/pkg/runtime"
	"errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/pkg/api/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"bytes"
	"fmt"
	"k8s.io/client-go/pkg/apis/batch/v2alpha1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var listErrorTests = []struct {
	resource string
}{
	{"secrets"},
	{"configmaps"},
	{"services"},
	{"deployments"},
	{"ingresses"},
	{"cronjobs"},
	{"persistentvolumeclaims"},
}

func TestKindService_CleanupKindWithErrorOnGetList(t *testing.T) {
	for _, entry := range listErrorTests {

		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, _, fakeClientSet := getKindService(t, config)

		var test testing_k8s.ReactionFunc

		test = func(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

			return true, nil, errors.New("explode")
		}

		fakeClientSet.PrependReactor("list", entry.resource, test)

		assert.EqualError(t, kindService.CleanupKind("foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

var deleteErrorTests = []struct {
	resource string
	list     runtime.Object
}{
	{"secrets", &v1.SecretList{Items: []v1.Secret{{ObjectMeta: meta.ObjectMeta{Name: "default-token-fff"}}, {ObjectMeta: meta.ObjectMeta{Name: "dummy"}}},}},
	{"configmaps", &v1.ConfigMapList{Items: []v1.ConfigMap{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"services", &v1.ServiceList{Items: []v1.Service{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"persistentvolumeclaims", &v1.PersistentVolumeClaimList{Items: []v1.PersistentVolumeClaim{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"deployments", &v1beta1.DeploymentList{Items: []v1beta1.Deployment{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"ingresses", &v1beta1.IngressList{Items: []v1beta1.Ingress{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"cronjobs", &v2alpha1.CronJobList{Items: []v2alpha1.CronJob{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
}

func TestKindService_CleanupKindWithErrorOnDeleteKind(t *testing.T) {
	for _, entry := range deleteErrorTests {
		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, _, fakeClientSet := getKindService(t, config)

		var list, deleteFunc testing_k8s.ReactionFunc

		list = func(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

			return true, entry.list, nil
		}

		deleteFunc = func(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

			return true, nil, errors.New("explode")
		}

		fakeClientSet.PrependReactor("list", entry.resource, list)
		fakeClientSet.PrependReactor("delete", entry.resource, deleteFunc)

		assert.EqualError(t, kindService.CleanupKind("foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

var deleteTests = []struct {
	resource string
	list     runtime.Object
	out string
}{
	{"secrets", &v1.SecretList{Items: []v1.Secret{{ObjectMeta: meta.ObjectMeta{Name: "default-token-fff"}}, {ObjectMeta: meta.ObjectMeta{Name: "dummy"}}},}, "Secret \"dummy\" was removed.\n"},
	{"configmaps", &v1.ConfigMapList{Items: []v1.ConfigMap{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "ConfigMap \"dummy\" was removed.\n"},
	{"services", &v1.ServiceList{Items: []v1.Service{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "Service \"dummy\" was removed.\n"},
	{"persistentvolumeclaims", &v1.PersistentVolumeClaimList{Items: []v1.PersistentVolumeClaim{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "PersistentVolumeClaim \"dummy\" was removed.\n"},
	{"deployments", &v1beta1.DeploymentList{Items: []v1beta1.Deployment{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "Deployment \"dummy\" was removed.\n"},
	{"ingresses", &v1beta1.IngressList{Items: []v1beta1.Ingress{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "Ingress \"dummy\" was removed.\n"},
	{"cronjobs", &v2alpha1.CronJobList{Items: []v2alpha1.CronJob{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "CronJob \"dummy\" was removed.\n"},
}

func TestKindService_CleanupKind(t *testing.T) {
	for _, entry := range deleteTests {
		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, _, fakeClientSet := getKindService(t, config)

		var list, deleteFunc testing_k8s.ReactionFunc

		list = func(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

			return true, entry.list, nil
		}

		deleteFunc = func(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

			return true, nil, nil
		}

		fakeClientSet.PrependReactor("list", entry.resource, list)
		fakeClientSet.PrependReactor("delete", entry.resource, deleteFunc)

		output := captureOutput(func() {
			assert.NoError(t, kindService.CleanupKind("foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func getKindService(t *testing.T, config loader.Config) (*kindService, *MockImagesInterface, *fake.Clientset) {
	imageServiceMock := new(MockImagesInterface)

	fakeClientSet := fake.NewSimpleClientset()

	return NewKind(fakeClientSet, imageServiceMock, config), imageServiceMock, fakeClientSet
}

func captureOutput(f func()) (string) {
	oldWriter := writer
	var buf bytes.Buffer
	defer func() {
		writer = oldWriter
	}()
	writer = &buf
	f()
	return buf.String()
}

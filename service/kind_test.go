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
	"kube-helper/model"
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

		fakeClientSet.PrependReactor("list", entry.resource, errorReturnFunc)

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

		fakeClientSet.PrependReactor("list", entry.resource, getObjectReturnFunc(entry.list))
		fakeClientSet.PrependReactor("delete", entry.resource, errorReturnFunc)

		assert.EqualError(t, kindService.CleanupKind("foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

var deleteTests = []struct {
	resource string
	list     runtime.Object
	out      string
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

		fakeClientSet.PrependReactor("list", entry.resource, getObjectReturnFunc(entry.list))
		fakeClientSet.PrependReactor("delete", entry.resource, nilReturnFunc)

		kindService.usedKind.secret = append(kindService.usedKind.secret, "foobarUsed")

		output := captureOutput(func() {
			assert.NoError(t, kindService.CleanupKind("foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
		assert.Len(t, fakeClientSet.Actions(), 8)
	}
}

func TestKindService_CleanupKindCronJobWithoutEnabledSupport(t *testing.T) {
	config := loader.Config{}

	kindService, _, fakeClientSet := getKindService(t, config)

	assert.NoError(t, kindService.CleanupKind("foobar"))
	assert.Len(t, fakeClientSet.Actions(), 6)

}

var secret = `kind: Secret
apiVersion: v1
type: Opaque
metadata:
  name: dummy`

var configMap = `kind: ConfigMap
apiVersion: v1
metadata:
  name: dummy`

var service = `kind: Service
apiVersion: v1
metadata:
  name: dummy`

var serviceWithAnnotation = `kind: Service
apiVersion: v1
metadata:
  name: dummy
  annotations:
    tourstream.eu/ingress: true`

var persistentVolumeClaim = `kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: dummy`

var persistentVolume = `kind: PersistentVolume
apiVersion: v1
metadata:
  name: dummy`

var deployment = `kind: Deployment
apiVersion: extensions/v1beta1
metadata:
  name: dummy`

var deploymentWithAnnotation = `kind: Deployment
apiVersion: extensions/v1beta1
metadata:
  name: dummy
  annotations:
    imageUpdateStrategy: "latest-branching"
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: deploy
        image: eu.gcr.io/foobar/app`

var ingress = `kind: Ingress
apiVersion: extensions/v1beta1
metadata:
  name: dummy`

var cronjob = `kind: CronJob
apiVersion: batch/v2alpha1
metadata:
  name: dummy`

var cronjobWithAnnotation = `kind: CronJob
apiVersion: batch/v2alpha1
metadata:
  name: dummy
  annotations:
    imageUpdateStrategy: "latest-branching"
spec:
  schedule: "*/30 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cron
            image: busy
          - name: cron-with-gcr
            image: eu.gcr.io/foobar/app`

var insertTests = []struct {
	resource string
	kind     string
	out      string
}{
	{"secrets", secret, "Secret \"dummy\" was generated.\n"},
	{"configmaps", configMap, "ConfigMap \"dummy\" was generated.\n"},
	{"services", service, "Service \"dummy\" was generated.\n"},
	{"persistentvolumeclaims", persistentVolumeClaim, "PersistentVolumeClaim \"dummy\" was generated.\n"},
	{"persistentvolumes", persistentVolume, "PersistentVolume \"dummy\" was generated.\n"},
	{"deployments", deployment, "Deployment \"dummy\" was generated.\n"},
	{"ingresses", ingress, "Ingress \"dummy\" was generated.\n"},
	{"cronjobs", cronjob, "CronJob \"dummy\" was generated.\n"},
}

var upsertTests = []struct {
	resource string
	kind     string
	out      string
	object   runtime.Object
}{
	{"secrets", secret, "Secret \"dummy\" was updated.\n", nil,},
	{"configmaps", configMap, "ConfigMap \"dummy\" was updated.\n", nil,},
	{"services", service, "Service \"dummy\" was updated.\n", &v1.Service{ObjectMeta: meta.ObjectMeta{Name: "dummy"}},},
	{"services", serviceWithAnnotation, "Service \"dummy\" was updated.\n", &v1.Service{ObjectMeta: meta.ObjectMeta{Name: "dummy"}},},
	{"persistentvolumeclaims", persistentVolumeClaim, "PersistentVolumeClaim \"dummy\" was updated.\n", &v1.PersistentVolumeClaim{ObjectMeta: meta.ObjectMeta{Name: "dummy"}},},
	{"persistentvolumes", persistentVolume, "PersistentVolume \"dummy\" was updated.\n", nil,},
	{"deployments", deployment, "Deployment \"dummy\" was updated.\n", nil,},
	{"ingresses", ingress, "Ingress \"dummy\" was updated.\n", nil,},
	{"cronjobs", cronjob, "CronJob \"dummy\" was updated.\n", nil,},
}

var setImageTests = []struct {
	resource string
	kind     string
	out      string
	object   runtime.Object
}{
	{"cronjobs", cronjobWithAnnotation, "CronJob \"dummy\" was updated.\n", nil,},
	{"deployments", deploymentWithAnnotation, "Deployment \"dummy\" was updated.\n", nil,},
}

func TestKindService_ApplyKindShouldFailWithErrorDuringDecode(t *testing.T) {
	kindService, _, _ := getKindService(t, loader.Config{})

	var kind = `kind: Pod2
apiVersion: v1
metadata:
  name: dummy`

	assert.EqualError(t, kindService.ApplyKind("foobar", []string{kind}, "foobar"), "no kind \"Pod2\" is registered for version \"v1\"")
}

func TestKindService_ApplyKindShouldFailWithInvalidKind(t *testing.T) {
	kindService, _, _ := getKindService(t, loader.Config{})

	var kind = `kind: Pod
apiVersion: v1
metadata:
  name: dummy`

	assert.EqualError(t, kindService.ApplyKind("foobar", []string{kind}, "foobar"), "Kind Pod is not supported.")
}

func TestKindService_ApplyKindInsertWithError(t *testing.T) {
	for _, entry := range insertTests {
		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, _, fakeClientSet := getKindService(t, config)
		fakeClientSet.PrependReactor("get", entry.resource, errorReturnFunc)
		fakeClientSet.PrependReactor("create", entry.resource, errorReturnFunc)

		assert.EqualError(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindInsert(t *testing.T) {
	for _, entry := range insertTests {
		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, _, fakeClientSet := getKindService(t, config)
		fakeClientSet.PrependReactor("get", entry.resource, errorReturnFunc)
		fakeClientSet.PrependReactor("create", entry.resource, nilReturnFunc)

		output := captureOutput(func() {
			assert.NoError(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindUpdateWithError(t *testing.T) {
	for _, entry := range upsertTests {
		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, _, fakeClientSet := getKindService(t, config)
		fakeClientSet.PrependReactor("get", entry.resource, getObjectReturnFunc(entry.object))
		fakeClientSet.PrependReactor("update", entry.resource, errorReturnFunc)

		assert.EqualError(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindUpdate(t *testing.T) {
	for _, entry := range upsertTests {
		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, _, fakeClientSet := getKindService(t, config)
		fakeClientSet.PrependReactor("get", entry.resource, getObjectReturnFunc(entry.object))
		fakeClientSet.PrependReactor("update", entry.resource, nilReturnFunc)

		output := captureOutput(func() {
			assert.NoError(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindUpdateWithContainers(t *testing.T) {
	for _, entry := range setImageTests {
		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, imageServiceMock, fakeClientSet := getKindService(t, config)
		fakeClientSet.PrependReactor("get", entry.resource, getObjectReturnFunc(entry.object))
		fakeClientSet.PrependReactor("update", entry.resource, nilReturnFunc)

		imageServiceMock.On("List", loader.Cleanup{ImagePath: "eu.gcr.io/foobar/app"}).Return(new(model.TagCollection), nil)

		output := captureOutput(func() {
			assert.NoError(t, kindService.ApplyKind("dummy-foobar2", []string{entry.kind}, "foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindUpdateWithContainersError(t *testing.T) {
	for _, entry := range setImageTests {
		config := loader.Config{
			Cluster: loader.Cluster{
				AlphaSupport: true,
			},
		}

		kindService, imageServiceMock, _ := getKindService(t, config)

		imageServiceMock.On("List", loader.Cleanup{ImagePath: "eu.gcr.io/foobar/app"}).Return(nil, errors.New("explode"))

		assert.Error(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))

	}
}

func TestKindService_ApplyKindUpdateWithDisabledAlphaSupport(t *testing.T) {

	kindService, _, _ := getKindService(t, loader.Config{})

	output := captureOutput(func() {
		assert.NoError(t, kindService.ApplyKind("foobar", []string{cronjob}, "foobar"))
	})

	assert.Equal(t, "CronJob \"dummy\" was not generated or updated, because alpha support is not enabled.\n", output)
}

func TestKindService_SetImageForContainer(t *testing.T) {
	var dataProvider = []struct {
		namespace string
		tags     []string
		imagePath      string
	}{
		{"foobar", []string{"staging-foobar-latest", "staging-foobar-3"}, "gcr.io/path/app:staging-foobar-3", },
		{"production", []string{"latest", "prod-3"}, "gcr.io/path/app:prod-3", },
		{"staging", []string{"staging-latest", "staging-3"}, "gcr.io/path/app:staging-3", },
	}


	for _, entry := range dataProvider {
		kindService, imageServiceMock, _ := getKindService(t, loader.Config{})

		tags := new(model.TagCollection)
		tags.Manifests = map[string]model.Manifest{}
		tags.Manifests["stuff"] = model.Manifest{
			Tags: entry.tags,
		}

		imageServiceMock.On("List", loader.Cleanup{ImagePath: "gcr.io/path/app"}).Return(tags, nil)

		containers := []v1.Container{
			{Image: "gcr.io/path/app"},
		}

		kindService.setImageForContainer("latest-branching", containers, entry.namespace)

		assert.Equal(t, entry.imagePath, containers[0].Image)
	}


}

func errorReturnFunc(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

	return true, nil, errors.New("explode")
}

func nilReturnFunc(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

	return true, nil, nil
}

func getObjectReturnFunc(obj runtime.Object) testing_k8s.ReactionFunc {
	return func(action testing_k8s.Action) (handled bool, ret runtime.Object, err error) {

		return true, obj, nil
	}
}

func getKindService(t *testing.T, config loader.Config) (*kindService, *MockImagesInterface, *fake.Clientset) {
	imageServiceMock := new(MockImagesInterface)

	fakeClientSet := fake.NewSimpleClientset()

	return newKind(fakeClientSet, imageServiceMock, config), imageServiceMock, fakeClientSet
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

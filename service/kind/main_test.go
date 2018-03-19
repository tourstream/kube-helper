package kind

import (
	"bytes"
	"errors"
	"fmt"
	"kube-helper/loader"
	"kube-helper/mocks"
	"kube-helper/model"
	"testing"

	testingKube "kube-helper/testing"

	"github.com/stretchr/testify/assert"
	apps "k8s.io/api/apps/v1beta2"
	batch "k8s.io/api/batch/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
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

		config := loader.Config{}

		kindService, _, fakeClientSet := getKindServiceInterface(config)

		fakeClientSet.PrependReactor("list", entry.resource, testingKube.ErrorReturnFunc)

		assert.EqualError(t, kindService.CleanupKind("foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

var deleteErrorTests = []struct {
	resource string
	list     runtime.Object
}{
	{"secrets", &coreV1.SecretList{Items: []coreV1.Secret{{ObjectMeta: meta.ObjectMeta{Name: "default-token-fff"}}, {ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"configmaps", &coreV1.ConfigMapList{Items: []coreV1.ConfigMap{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"services", &coreV1.ServiceList{Items: []coreV1.Service{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"persistentvolumeclaims", &coreV1.PersistentVolumeClaimList{Items: []coreV1.PersistentVolumeClaim{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"deployments", &apps.DeploymentList{Items: []apps.Deployment{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"ingresses", &extensions.IngressList{Items: []extensions.Ingress{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
	{"cronjobs", &batch.CronJobList{Items: []batch.CronJob{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}},
}

func TestKindService_CleanupKindWithErrorOnDeleteKind(t *testing.T) {
	for _, entry := range deleteErrorTests {
		config := loader.Config{}

		kindService, _, fakeClientSet := getKindServiceInterface(config)

		fakeClientSet.PrependReactor("list", entry.resource, testingKube.GetObjectReturnFunc(entry.list))
		fakeClientSet.PrependReactor("delete", entry.resource, testingKube.ErrorReturnFunc)

		assert.EqualError(t, kindService.CleanupKind("foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

var deleteTests = []struct {
	resource string
	list     runtime.Object
	out      string
}{
	{"secrets", &coreV1.SecretList{Items: []coreV1.Secret{{ObjectMeta: meta.ObjectMeta{Name: "default-token-fff"}}, {ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "Secret \"dummy\" was removed.\n"},
	{"configmaps", &coreV1.ConfigMapList{Items: []coreV1.ConfigMap{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "ConfigMap \"dummy\" was removed.\n"},
	{"services", &coreV1.ServiceList{Items: []coreV1.Service{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "Service \"dummy\" was removed.\n"},
	{"persistentvolumeclaims", &coreV1.PersistentVolumeClaimList{Items: []coreV1.PersistentVolumeClaim{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "PersistentVolumeClaim \"dummy\" was removed.\n"},
	{"deployments", &apps.DeploymentList{Items: []apps.Deployment{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "Deployment \"dummy\" was removed.\n"},
	{"ingresses", &extensions.IngressList{Items: []extensions.Ingress{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "Ingress \"dummy\" was removed.\n"},
	{"cronjobs", &batch.CronJobList{Items: []batch.CronJob{{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}}}, "CronJob \"dummy\" was removed.\n"},
}

func TestKindService_CleanupKind(t *testing.T) {
	for _, entry := range deleteTests {
		config := loader.Config{}

		kindService, _, fakeClientSet := getKindService(config)

		fakeClientSet.PrependReactor("list", entry.resource, testingKube.GetObjectReturnFunc(entry.list))
		fakeClientSet.PrependReactor("delete", entry.resource, testingKube.NilReturnFunc)

		kindService.usedKind.secret = append(kindService.usedKind.secret, "foobarUsed")

		output := captureOutput(func() {
			assert.NoError(t, kindService.CleanupKind("foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
		assert.Len(t, fakeClientSet.Actions(), 8)
	}
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
    "tourstream.eu/ingress": "true"`

var persistentVolumeClaim = `kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: dummy`

var persistentVolume = `kind: PersistentVolume
apiVersion: v1
metadata:
  name: dummy`

var deployment = `kind: Deployment
apiVersion: apps/v1beta2
metadata:
  name: dummy`

var deploymentWithAnnotation = `kind: Deployment
apiVersion: apps/v1beta2
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
apiVersion: batch/v1beta1
metadata:
  name: dummy`

var cronjobWithAnnotation = `kind: CronJob
apiVersion: batch/v1beta1
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
	{"secrets", secret, "Secret \"dummy\" was updated.\n", nil},
	{"configmaps", configMap, "ConfigMap \"dummy\" was updated.\n", nil},
	{"services", service, "Service \"dummy\" was updated.\n", &coreV1.Service{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}},
	{"services", serviceWithAnnotation, "Service \"dummy\" was updated.\n", &coreV1.Service{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}},
	{"persistentvolumeclaims", persistentVolumeClaim, "PersistentVolumeClaim \"dummy\" was updated.\n", &coreV1.PersistentVolumeClaim{ObjectMeta: meta.ObjectMeta{Name: "dummy"}}},
	{"persistentvolumes", persistentVolume, "PersistentVolume \"dummy\" was updated.\n", nil},
	{"deployments", deployment, "Deployment \"dummy\" was updated.\n", nil},
	{"ingresses", ingress, "Ingress \"dummy\" was updated.\n", nil},
	{"cronjobs", cronjob, "CronJob \"dummy\" was updated.\n", nil},
}

var setImageTests = []struct {
	resource string
	kind     string
	out      string
	object   runtime.Object
}{
	{"cronjobs", cronjobWithAnnotation, "CronJob \"dummy\" was updated.\n", nil},
	{"deployments", deploymentWithAnnotation, "Deployment \"dummy\" was updated.\n", nil},
}

func TestKindService_ApplyKindShouldFailWithErrorDuringDecode(t *testing.T) {
	kindService, _, _ := getKindServiceInterface(loader.Config{})

	var kind = `kind: Pod2
apiVersion: v1
metadata:
  name: dummy`

	assert.EqualError(t, kindService.ApplyKind("foobar", []string{kind}, "foobar"), "no kind \"Pod2\" is registered for version \"v1\"")
}

func TestKindService_ApplyKindShouldFailWithInvalidKind(t *testing.T) {
	kindService, _, _ := getKindServiceInterface(loader.Config{})

	var kind = `kind: Pod
apiVersion: v1
metadata:
  name: dummy`

	assert.EqualError(t, kindService.ApplyKind("foobar", []string{kind}, "foobar"), "kind Pod is not supported")
}

func TestKindService_ApplyKindInsertWithError(t *testing.T) {
	for _, entry := range insertTests {
		config := loader.Config{}

		kindService, _, fakeClientSet := getKindServiceInterface(config)
		fakeClientSet.PrependReactor("get", entry.resource, testingKube.ErrorReturnFunc)
		fakeClientSet.PrependReactor("create", entry.resource, testingKube.ErrorReturnFunc)

		assert.EqualError(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindInsert(t *testing.T) {
	for _, entry := range insertTests {
		config := loader.Config{}

		kindService, _, fakeClientSet := getKindServiceInterface(config)
		fakeClientSet.PrependReactor("get", entry.resource, testingKube.ErrorReturnFunc)
		fakeClientSet.PrependReactor("create", entry.resource, testingKube.NilReturnFunc)

		output := captureOutput(func() {
			assert.NoError(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindUpdateWithError(t *testing.T) {
	for _, entry := range upsertTests {
		config := loader.Config{}

		kindService, _, fakeClientSet := getKindServiceInterface(config)
		fakeClientSet.PrependReactor("get", entry.resource, testingKube.GetObjectReturnFunc(entry.object))
		fakeClientSet.PrependReactor("update", entry.resource, testingKube.ErrorReturnFunc)

		assert.EqualError(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), "explode", fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindUpdate(t *testing.T) {
	for _, entry := range upsertTests {
		config := loader.Config{}

		kindService, _, fakeClientSet := getKindServiceInterface(config)
		fakeClientSet.PrependReactor("get", entry.resource, testingKube.GetObjectReturnFunc(entry.object))
		fakeClientSet.PrependReactor("update", entry.resource, testingKube.NilReturnFunc)

		output := captureOutput(func() {
			assert.NoError(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindUpdateWithContainers(t *testing.T) {
	for _, entry := range setImageTests {
		config := loader.Config{}

		kindService, imageServiceMock, fakeClientSet := getKindServiceInterface(config)
		fakeClientSet.PrependReactor("get", entry.resource, testingKube.GetObjectReturnFunc(entry.object))
		fakeClientSet.PrependReactor("update", entry.resource, testingKube.NilReturnFunc)

		imageServiceMock.On("List", loader.Cleanup{ImagePath: "eu.gcr.io/foobar/app"}).Return(new(model.TagCollection), nil)

		output := captureOutput(func() {
			assert.NoError(t, kindService.ApplyKind("dummy-foobar2", []string{entry.kind}, "foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))
		})

		assert.Equal(t, entry.out, output, fmt.Sprintf("Test failed for resource %s", entry.resource))
	}
}

func TestKindService_ApplyKindUpdateWithContainersError(t *testing.T) {
	for _, entry := range setImageTests {
		config := loader.Config{}

		kindService, imageServiceMock, _ := getKindServiceInterface(config)

		imageServiceMock.On("List", loader.Cleanup{ImagePath: "eu.gcr.io/foobar/app"}).Return(nil, errors.New("explode"))

		assert.Error(t, kindService.ApplyKind("foobar", []string{entry.kind}, "foobar"), fmt.Sprintf("Test failed for resource %s", entry.resource))

	}
}

func TestKindService_SetImageForContainer(t *testing.T) {
	var dataProvider = []struct {
		namespace string
		tags      []string
		imagePath string
		config    loader.Config
	}{
		{"foobar", []string{"staging-foobar-latest", "staging-foobar-3"}, "gcr.io/path/app:staging-foobar-3", loader.Config{}},
		{"production", []string{"latest", "prod-3"}, "gcr.io/path/app:prod-3", loader.Config{Internal: loader.Internal{IsProduction: true}}},
		{"random", []string{"latest", "prod-3"}, "gcr.io/path/app:prod-3", loader.Config{Internal: loader.Internal{IsProduction: true}}},
		{"staging", []string{"staging-latest", "staging-3"}, "gcr.io/path/app:staging-3", loader.Config{}},
	}

	for _, entry := range dataProvider {
		kindService, imageServiceMock, _ := getKindService(entry.config)

		tags := new(model.TagCollection)
		tags.Manifests = map[string]model.Manifest{}
		tags.Manifests["stuff"] = model.Manifest{
			Tags: entry.tags,
		}

		imageServiceMock.On("List", loader.Cleanup{ImagePath: "gcr.io/path/app"}).Return(tags, nil)

		containers := []coreV1.Container{
			{Image: "gcr.io/path/app"},
		}

		kindService.setImageForContainer(map[string]string{"imageUpdateStrategy": "latest-branching"}, containers, entry.namespace)

		assert.Equal(t, entry.imagePath, containers[0].Image)
	}

}

func getKindServiceInterface(config loader.Config) (KindInterface, *mocks.ImagesInterface, *fake.Clientset) {
	imageServiceMock := new(mocks.ImagesInterface)

	fakeClientSet := fake.NewSimpleClientset()

	return NewKind(fakeClientSet, imageServiceMock, config), imageServiceMock, fakeClientSet
}

func getKindService(config loader.Config) (*kindService, *mocks.ImagesInterface, *fake.Clientset) {

	imageServiceMock := new(mocks.ImagesInterface)

	fakeClientSet := fake.NewSimpleClientset()

	k := new(kindService)
	k.clientSet = fakeClientSet
	k.imagesService = imageServiceMock
	k.config = config
	k.usedKind = usedKind{}
	k.decoder = scheme.Codecs.UniversalDeserializer()

	return k, imageServiceMock, fakeClientSet
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

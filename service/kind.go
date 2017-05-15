package service

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/batch/v2alpha1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
)

type KindInterface interface {
	createKind(kubernetesNamespace string, fileLines []string) error
}

type kindService struct {
	decoder runtime.Decoder
	client  kubernetes.Interface
}

func NewKind(client kubernetes.Interface) *kindService {
	k := new(kindService)
	k.decoder = api.Codecs.UniversalDecoder(unversioned.GroupVersion{
		Version: "v1",
	}, unversioned.GroupVersion{
		Group:   "extensions",
		Version: "v1beta1",
	}, unversioned.GroupVersion{
		Group:   "batch",
		Version: "v2alpha1",
	})
	k.client = client

	return k
}

func (k *kindService) CreateKind(kubernetesNamespace string, fileLines []string) error {
	fileContent, _, err := k.decoder.Decode([]byte(strings.Join(fileLines, "\n")), nil, nil)
	if err != nil {
		return err
	}
	switch fileContent.GetObjectKind().GroupVersionKind().Kind {
	case "Secret":
		return k.createSecrets(kubernetesNamespace, fileContent.(*v1.Secret))
	case "ConfigMap":
		return k.createConfigMap(kubernetesNamespace, fileContent.(*v1.ConfigMap))
	case "Service":
		return k.createService(kubernetesNamespace, fileContent.(*v1.Service))
	case "Deployment":
		return k.createDeployment(kubernetesNamespace, fileContent.(*v1beta1.Deployment))
	case "Ingress":
		return k.createIngress(kubernetesNamespace, fileContent.(*v1beta1.Ingress))
	case "CronJob":
		return k.createCronJob(kubernetesNamespace, fileContent.(*v2alpha1.CronJob))
	default:
		return errors.New(fmt.Sprintf("Kind %s is not supported.", fileContent.GetObjectKind().GroupVersionKind().Kind))
	}

}

func (k *kindService) createCronJob(kubernetesNamespace string, cronJob *v2alpha1.CronJob) error {

	_, err := k.client.BatchV2alpha1().CronJobs(kubernetesNamespace).Create(cronJob)

	if err != nil {
		return err
	}

	log.Printf("CronJob \"%s\" was generated\n", cronJob.ObjectMeta.Name)

	return nil
}

func (k *kindService) createDeployment(kubernetesNamespace string, deployment *v1beta1.Deployment) error {

	_, err := k.client.ExtensionsV1beta1().Deployments(kubernetesNamespace).Create(deployment)

	if err != nil {
		return err
	}

	log.Printf("Deployment \"%s\" was generated\n", deployment.ObjectMeta.Name)

	return nil
}

func (k *kindService) createIngress(kubernetesNamespace string, ingress *v1beta1.Ingress) error {

	_, err := k.client.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Create(ingress)

	if err != nil {
		return err
	}

	log.Printf("Ingress \"%s\" was generated\n", ingress.ObjectMeta.Name)

	return nil
}

func (k *kindService) createService(kubernetesNamespace string, service *v1.Service) error {
	_, err := k.client.CoreV1().Services(kubernetesNamespace).Create(service)

	if err != nil {
		return err
	}

	log.Printf("Service \"%s\" was generated\n", service.ObjectMeta.Name)

	return nil
}

func (k *kindService) createSecrets(kubernetesNamespace string, secret *v1.Secret) error {
	_, err := k.client.CoreV1().Secrets(kubernetesNamespace).Create(secret)

	if err != nil {
		return err
	}

	log.Printf("Secret \"%s\" was generated\n", secret.ObjectMeta.Name)

	return nil
}

func (k *kindService) createConfigMap(kubernetesNamespace string, configMap *v1.ConfigMap) error {
	_, err := k.client.CoreV1().ConfigMaps(kubernetesNamespace).Create(configMap)

	if err != nil {
		return err
	}

	log.Printf("ConfigMap \"%s\" was generated\n", configMap.ObjectMeta.Name)

	return nil
}

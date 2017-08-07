package service

import (
	"errors"
	"fmt"
	"log"
	"strings"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/batch/v2alpha1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"kube-helper/loader"
	"kube-helper/util"
)

type KindInterface interface {
	CreateKind(kubernetesNamespace string, fileLines []string) error
	UpdateKind(kubernetesNamespace string, fileLines []string) error
}

type kindService struct {
	decoder       runtime.Decoder
	clientSet     kubernetes.Interface
	imagesService ImagesInterface
	config        loader.Config
}

func NewKind(client kubernetes.Interface, imagesService ImagesInterface, config loader.Config) *kindService {
	k := new(kindService)
	k.decoder = api.Codecs.UniversalDecoder(schema.GroupVersion{
		Version: "v1",
	}, schema.GroupVersion{
		Group:   "extensions",
		Version: "v1beta1",
	}, schema.GroupVersion{
		Group:   "batch",
		Version: "v2alpha1",
	})
	k.clientSet = client
	k.imagesService = imagesService
	k.config = config

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

func (k *kindService) UpdateKind(kubernetesNamespace string, fileLines []string) error {
	fileContent, _, err := k.decoder.Decode([]byte(strings.Join(fileLines, "\n")), nil, nil)
	if err != nil {
		return err
	}

	switch fileContent.GetObjectKind().GroupVersionKind().Kind {
	case "Secret":
		return k.updateSecrets(kubernetesNamespace, fileContent.(*v1.Secret))
	case "ConfigMap":
		return k.updateConfigMap(kubernetesNamespace, fileContent.(*v1.ConfigMap))
	case "Service":
		log.Print("Service update is not supported.")
	case "Deployment":
		return k.updateDeployment(kubernetesNamespace, fileContent.(*v1beta1.Deployment))
	case "Ingress":
		log.Print("Ingress update is not supported.")
	case "CronJob":
		return k.updateCronJob(kubernetesNamespace, fileContent.(*v2alpha1.CronJob))
	default:
		return errors.New(fmt.Sprintf("Kind %s is not supported.", fileContent.GetObjectKind().GroupVersionKind().Kind))
	}
	return nil
}

func (k *kindService) updateCronJob(kubernetesNamespace string, cronJob *v2alpha1.CronJob) error {

	if _, ok := cronJob.Annotations["imageUpdateStrategy"]; ok {
		err := k.setImageForContainer(cronJob.Annotations["imageUpdateStrategy"], cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, kubernetesNamespace)

		if err != nil {
			return err
		}
	}

	_, err := k.clientSet.BatchV2alpha1().CronJobs(kubernetesNamespace).Get(cronJob.Name, meta_v1.GetOptions{})

	if err != nil {
		err = k.createCronJob(kubernetesNamespace, cronJob)
		if err != nil {
			return err
		}

	}
	_, err = k.clientSet.BatchV2alpha1().CronJobs(kubernetesNamespace).Update(cronJob)

	if err != nil {
		return err
	}

	log.Printf("CronJob \"%s\" was updated\n", cronJob.ObjectMeta.Name)

	return nil
}

func (k *kindService) updateDeployment(kubernetesNamespace string, deployment *v1beta1.Deployment) error {

	if _, ok := deployment.Annotations["imageUpdateStrategy"]; ok {
		err := k.setImageForContainer(deployment.Annotations["imageUpdateStrategy"], deployment.Spec.Template.Spec.Containers, kubernetesNamespace)

		if err != nil {
			return err
		}
	}

	_, err := k.clientSet.ExtensionsV1beta1().Deployments(kubernetesNamespace).Update(deployment)

	if err != nil {
		return err
	}

	log.Printf("Deployment \"%s\" was updated\n", deployment.ObjectMeta.Name)

	return nil
}

func (k *kindService) updateSecrets(kubernetesNamespace string, secret *v1.Secret) error {
	_, err := k.clientSet.CoreV1().Secrets(kubernetesNamespace).Update(secret)

	if err != nil {
		return err
	}

	log.Printf("Secret \"%s\" was updated\n", secret.ObjectMeta.Name)

	return nil
}

func (k *kindService) updateConfigMap(kubernetesNamespace string, configMap *v1.ConfigMap) error {
	_, err := k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).Update(configMap)

	if err != nil {
		return err
	}

	log.Printf("ConfigMap \"%s\" was updated\n", configMap.ObjectMeta.Name)

	return nil
}

func (k *kindService) createCronJob(kubernetesNamespace string, cronJob *v2alpha1.CronJob) error {

	if _, ok := cronJob.Annotations["imageUpdateStrategy"]; ok {
		err := k.setImageForContainer(cronJob.Annotations["imageUpdateStrategy"], cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, kubernetesNamespace)

		if err != nil {
			return err
		}
	}

	_, err := k.clientSet.BatchV2alpha1().CronJobs(kubernetesNamespace).Create(cronJob)

	if err != nil {
		return err
	}

	log.Printf("CronJob \"%s\" was generated\n", cronJob.ObjectMeta.Name)

	return nil
}

func (k *kindService) createDeployment(kubernetesNamespace string, deployment *v1beta1.Deployment) error {

	if _, ok := deployment.Annotations["imageUpdateStrategy"]; ok {
		err := k.setImageForContainer(deployment.Annotations["imageUpdateStrategy"], deployment.Spec.Template.Spec.Containers, kubernetesNamespace)

		if err != nil {
			return err
		}
	}

	_, err := k.clientSet.ExtensionsV1beta1().Deployments(kubernetesNamespace).Create(deployment)

	if err != nil {
		return err
	}

	log.Printf("Deployment \"%s\" was generated\n", deployment.ObjectMeta.Name)

	return nil
}

func (k *kindService) createIngress(kubernetesNamespace string, ingress *v1beta1.Ingress) error {

	_, err := k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Create(ingress)

	if err != nil {
		return err
	}

	log.Printf("Ingress \"%s\" was generated\n", ingress.ObjectMeta.Name)

	return nil
}

func (k *kindService) createService(kubernetesNamespace string, service *v1.Service) error {
	_, err := k.clientSet.CoreV1().Services(kubernetesNamespace).Create(service)

	if err != nil {
		return err
	}

	log.Printf("Service \"%s\" was generated\n", service.ObjectMeta.Name)

	return nil
}

func (k *kindService) createSecrets(kubernetesNamespace string, secret *v1.Secret) error {
	_, err := k.clientSet.CoreV1().Secrets(kubernetesNamespace).Create(secret)

	if err != nil {
		return err
	}

	log.Printf("Secret \"%s\" was generated\n", secret.ObjectMeta.Name)

	return nil
}

func (k *kindService) createConfigMap(kubernetesNamespace string, configMap *v1.ConfigMap) error {
	_, err := k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).Create(configMap)

	if err != nil {
		return err
	}

	log.Printf("ConfigMap \"%s\" was generated\n", configMap.ObjectMeta.Name)

	return nil
}

func (k *kindService) setImageForContainer(strategy string, containers []v1.Container, kubernetesNamespace string) error {

	var imagesService ImagesInterface = new(Images)

	for idx, container := range containers {

		if strings.Contains(container.Image, "gcr.io") == false {
			continue
		}

		images, err := imagesService.List(loader.Cleanup{ImagePath: container.Image})

		if err != nil {
			return err
		}

		switch strategy {
		case "latest-branching":

			latestTag := "staging-" + kubernetesNamespace + "-latest"

			if kubernetesNamespace == "staging" {
				latestTag = "staging-latest"
			}

			tag := getVersionForLatestTag(latestTag, images)

			if tag != "" {
				containers[idx].Image += ":" + tag
			}
		}
	}

	return nil
}

func getVersionForLatestTag(latestTag string, images *TagCollection) string {
	for _, manifest := range images.Manifests {
		if util.InArray(manifest.Tags, latestTag) {
			for _, tag := range manifest.Tags {
				if tag != latestTag && tag != "latest" {
					return tag
				}
			}
		}
	}

	return ""
}

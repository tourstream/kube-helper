package service

import (
	"errors"
	"fmt"
	"strings"

	"kube-helper/loader"
	"kube-helper/util"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/batch/v2alpha1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"kube-helper/model"
	"os"
	"io"
)

var writer io.Writer = os.Stdout

type KindInterface interface {
	ApplyKind(kubernetesNamespace string, fileLines []string) error
	CleanupKind(kubernetesNamespace string, fileLines []string) error
}

type usedKind struct {
	secret                []string
	cronJob               []string
	deployment            []string
	service               []string
	ingress               []string
	configMap             []string
	persistentVolume      []string
	persistentVolumeClaim []string
}

type kindService struct {
	decoder       runtime.Decoder
	clientSet     kubernetes.Interface
	imagesService ImagesInterface
	config        loader.Config
	usedKind      usedKind
}

func NewKind(client kubernetes.Interface, imagesService ImagesInterface, config loader.Config) *kindService {
	k := kindService{clientsetscheme.Codecs.UniversalDeserializer(), client, imagesService, config, usedKind{}}
	return &k
}

func (k *kindService) CleanupKind(kubernetesNamespace string) error {

	err := k.cleanupSecret(kubernetesNamespace)

	if err != nil {
		return err
	}

	err = k.cleanupConfigMaps(kubernetesNamespace)

	if err != nil {
		return err
	}

	err = k.cleanupCronjobs(kubernetesNamespace)

	if err != nil {
		return err
	}

	err = k.cleanupDeployment(kubernetesNamespace)

	if err != nil {
		return err
	}

	err = k.cleanupIngresses(kubernetesNamespace)

	if err != nil {
		return err
	}

	err = k.cleanupPersistentVolumeClaims(kubernetesNamespace)

	if err != nil {
		return err
	}

	return k.cleanupServices(kubernetesNamespace)
}

func (k *kindService) cleanupSecret(kubernetesNamespace string) error {
	list, err := k.clientSet.CoreV1().Secrets(kubernetesNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	names := []string{}

	for _, listEntry := range list.Items {
		if strings.HasPrefix(listEntry.Name, "default-token-") {
			continue
		}
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.secret) {
		err = k.clientSet.CoreV1().Secrets(kubernetesNamespace).Delete(name, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}
		fmt.Fprintf(writer, "Secret \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupConfigMaps(kubernetesNamespace string) error {
	list, err := k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	names := []string{}

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.configMap) {
		err = k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).Delete(name, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "ConfigMap \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupServices(kubernetesNamespace string) error {
	list, err := k.clientSet.CoreV1().Services(kubernetesNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	names := []string{}

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.service) {
		err = k.clientSet.CoreV1().Services(kubernetesNamespace).Delete(name, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Service \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupDeployment(kubernetesNamespace string) error {
	list, err := k.clientSet.ExtensionsV1beta1().Deployments(kubernetesNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	names := []string{}

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.deployment) {
		err = k.clientSet.ExtensionsV1beta1().Deployments(kubernetesNamespace).Delete(name, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Deployment \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupIngresses(kubernetesNamespace string) error {
	list, err := k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	names := []string{}

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.ingress) {
		err = k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Delete(name, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Ingress \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupCronjobs(kubernetesNamespace string) error {

	if !k.config.Cluster.AlphaSupport {
		return nil
	}

	list, err := k.clientSet.BatchV2alpha1().CronJobs(kubernetesNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	names := []string{}

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.cronJob) {
		err = k.clientSet.BatchV2alpha1().CronJobs(kubernetesNamespace).Delete(name, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "CronJob \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupPersistentVolumeClaims(kubernetesNamespace string) error {
	list, err := k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).List(meta_v1.ListOptions{})

	if err != nil {
		return err
	}

	names := []string{}

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.persistentVolumeClaim) {
		err = k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).Delete(name, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "PersistentVolumeClaim \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) ApplyKind(kubernetesNamespace string, fileLines []string) error {

	fileContent, _, err := k.decoder.Decode([]byte(strings.Join(fileLines, "\n")), nil, nil)

	if err != nil {
		return err
	}
	switch fileContent.GetObjectKind().GroupVersionKind().Kind {
	case "Secret":
		return k.upsertSecrets(kubernetesNamespace, fileContent.(*v1.Secret))
	case "ConfigMap":
		return k.upsertConfigMap(kubernetesNamespace, fileContent.(*v1.ConfigMap))
	case "Service":
		return k.upsertService(kubernetesNamespace, fileContent.(*v1.Service))
	case "Deployment":
		return k.upsertDeployment(kubernetesNamespace, fileContent.(*v1beta1.Deployment))
	case "Ingress":
		return k.upsertIngress(kubernetesNamespace, fileContent.(*v1beta1.Ingress))
	case "CronJob":
		return k.upsertCronJob(kubernetesNamespace, fileContent.(*v2alpha1.CronJob))
	case "PersistentVolume":
		return k.upsertPersistentVolume(fileContent.(*v1.PersistentVolume))
	case "PersistentVolumeClaim":
		return k.upsertPersistentVolumeClaim(kubernetesNamespace, fileContent.(*v1.PersistentVolumeClaim))
	default:
		return errors.New(fmt.Sprintf("Kind %s is not supported.", fileContent.GetObjectKind().GroupVersionKind().Kind))
	}
}

func (k *kindService) upsertSecrets(kubernetesNamespace string, secret *v1.Secret) error {
	_, err := k.clientSet.CoreV1().Secrets(kubernetesNamespace).Get(secret.Name, meta_v1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.CoreV1().Secrets(kubernetesNamespace).Create(secret)

		if err != nil {
			return err
		}

		k.usedKind.secret = append(k.usedKind.secret, secret.Name)

		fmt.Fprintf(writer, "Secret \"%s\" was generated.\n", secret.Name)

		return nil
	}

	_, err = k.clientSet.CoreV1().Secrets(kubernetesNamespace).Update(secret)

	if err != nil {
		return err
	}

	k.usedKind.secret = append(k.usedKind.secret, secret.Name)

	fmt.Fprintf(writer, "Secret \"%s\" was updated.\n", secret.Name)

	return nil
}

func (k *kindService) upsertCronJob(kubernetesNamespace string, cronJob *v2alpha1.CronJob) error {

	if !k.config.Cluster.AlphaSupport {
		return nil
	}

	if _, ok := cronJob.Annotations["imageUpdateStrategy"]; ok {
		err := k.setImageForContainer(cronJob.Annotations["imageUpdateStrategy"], cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, kubernetesNamespace)

		if err != nil {
			return err
		}
	}

	_, err := k.clientSet.BatchV2alpha1().CronJobs(kubernetesNamespace).Get(cronJob.Name, meta_v1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.BatchV2alpha1().CronJobs(kubernetesNamespace).Create(cronJob)

		if err != nil {
			return err
		}

		k.usedKind.cronJob = append(k.usedKind.cronJob, cronJob.Name)

		fmt.Fprintf(writer, "CronJob \"%s\" was generated.\n", cronJob.Name)

		return nil
	}
	_, err = k.clientSet.BatchV2alpha1().CronJobs(kubernetesNamespace).Update(cronJob)

	if err != nil {
		return err
	}

	k.usedKind.cronJob = append(k.usedKind.cronJob, cronJob.Name)

	fmt.Fprintf(writer, "CronJob \"%s\" was updated.\n", cronJob.Name)

	return nil
}

func (k *kindService) upsertDeployment(kubernetesNamespace string, deployment *v1beta1.Deployment) error {

	if _, ok := deployment.Annotations["imageUpdateStrategy"]; ok {
		err := k.setImageForContainer(deployment.Annotations["imageUpdateStrategy"], deployment.Spec.Template.Spec.Containers, kubernetesNamespace)

		if err != nil {
			return err
		}
	}

	_, err := k.clientSet.AppsV1beta1().Deployments(kubernetesNamespace).Get(deployment.Name, meta_v1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.ExtensionsV1beta1().Deployments(kubernetesNamespace).Create(deployment)

		if err != nil {
			return err
		}

		k.usedKind.deployment = append(k.usedKind.deployment, deployment.Name)

		fmt.Fprintf(writer, "Deployment \"%s\" was generated.\n", deployment.Name)

		return nil
	}

	_, err = k.clientSet.ExtensionsV1beta1().Deployments(kubernetesNamespace).Update(deployment)

	if err != nil {
		return err
	}

	k.usedKind.deployment = append(k.usedKind.deployment, deployment.Name)

	fmt.Fprintf(writer, "Deployment \"%s\" was updated.\n", deployment.Name)

	return nil
}

func (k *kindService) upsertService(kubernetesNamespace string, service *v1.Service) error {

	existingService, err := k.clientSet.CoreV1().Services(kubernetesNamespace).Get(service.Name, meta_v1.GetOptions{})

	if err != nil {

		_, err := k.clientSet.CoreV1().Services(kubernetesNamespace).Create(service)

		if err != nil {
			return err
		}

		k.usedKind.service = append(k.usedKind.service, service.Name)

		fmt.Fprintf(writer, "Service \"%s\" was generated.\n", service.Name)

		return nil
	}

	service.ResourceVersion = existingService.ResourceVersion
	service.Spec.ClusterIP = existingService.Spec.ClusterIP

	if _, ok := service.Annotations["tourstream.eu/ingress"]; ok {
		//TODO add better check which port is which, for now take the same ports like before so that the backend still works with it
		service.Spec.Ports = existingService.Spec.Ports
	}

	_, err = k.clientSet.CoreV1().Services(kubernetesNamespace).Update(service)

	if err != nil {
		return err
	}

	k.usedKind.service = append(k.usedKind.service, service.Name)

	fmt.Fprintf(writer, "Service \"%s\" was updated.\n", service.Name)

	return nil
}

func (k *kindService) upsertConfigMap(kubernetesNamespace string, configMap *v1.ConfigMap) error {

	_, err := k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).Get(configMap.Name, meta_v1.GetOptions{})

	if err != nil {

		_, err := k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).Create(configMap)

		if err != nil {
			return err
		}

		k.usedKind.configMap = append(k.usedKind.configMap, configMap.Name)

		fmt.Fprintf(writer, "ConfigMap \"%s\" was generated.\n", configMap.Name)

		return nil
	}

	_, err = k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).Update(configMap)

	if err != nil {
		return err
	}

	k.usedKind.configMap = append(k.usedKind.configMap, configMap.Name)

	fmt.Fprintf(writer, "ConfigMap \"%s\" was updated.\n", configMap.Name)

	return nil
}

func (k *kindService) upsertPersistentVolume(persistentVolume *v1.PersistentVolume) error {

	_, err := k.clientSet.CoreV1().PersistentVolumes().Get(persistentVolume.Name, meta_v1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.CoreV1().PersistentVolumes().Create(persistentVolume)

		if err != nil {
			return err
		}

		k.usedKind.persistentVolume = append(k.usedKind.persistentVolume, persistentVolume.Name)

		fmt.Fprintf(writer, "PersistentVolume \"%s\" was generated.\n", persistentVolume.Name)

		return nil
	}

	_, err = k.clientSet.CoreV1().PersistentVolumes().Update(persistentVolume)

	if err != nil {
		return err
	}

	k.usedKind.persistentVolume = append(k.usedKind.persistentVolume, persistentVolume.Name)

	fmt.Fprintf(writer, "PersistentVolume \"%s\" was updated.\n", persistentVolume.Name)

	return nil
}

func (k *kindService) upsertPersistentVolumeClaim(kubernetesNamespace string, persistentVolumeClaim *v1.PersistentVolumeClaim) error {

	existingClaim, err := k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).Get(persistentVolumeClaim.Name, meta_v1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).Create(persistentVolumeClaim)

		if err != nil {
			return err
		}

		k.usedKind.persistentVolumeClaim = append(k.usedKind.persistentVolumeClaim, persistentVolumeClaim.Name)

		fmt.Fprintf(writer, "PersistentVolumeClaim \"%s\" was generated.\n", persistentVolumeClaim.Name)

		return nil
	}

	//TODO check if change and fail in favour of a change in spec
	//override with existing spec because spec is immutable
	persistentVolumeClaim.Spec = existingClaim.Spec

	_, err = k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).Update(persistentVolumeClaim)

	if err != nil {
		return err
	}

	k.usedKind.persistentVolumeClaim = append(k.usedKind.persistentVolumeClaim, persistentVolumeClaim.Name)

	fmt.Fprintf(writer, "PersistentVolumeClaim \"%s\" was updated.\n", persistentVolumeClaim.Name)

	return nil
}

func (k *kindService) upsertIngress(kubernetesNamespace string, ingress *v1beta1.Ingress) error {

	_, err := k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Get(ingress.Name, meta_v1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Create(ingress)

		if err != nil {
			return err
		}

		k.usedKind.ingress = append(k.usedKind.ingress, ingress.Name)

		fmt.Fprintf(writer, "Ingress \"%s\" was generated.\n", ingress.Name)

		return nil
	}

	_, err = k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Update(ingress)

	if err != nil {
		return err
	}

	k.usedKind.ingress = append(k.usedKind.ingress, ingress.Name)

	fmt.Fprintf(writer, "Ingress \"%s\" was updated.\n", ingress.Name)

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

			latestTag := loader.StagingEnvironment + "-" + kubernetesNamespace + "-latest"

			if kubernetesNamespace == loader.StagingEnvironment {
				latestTag = "staging-latest"
			}

			if kubernetesNamespace == loader.ProductionEnvironment {
				latestTag = "latest"
			}

			tag := getVersionForLatestTag(latestTag, images)

			if tag != "" {
				containers[idx].Image += ":" + tag
			}
		}
	}

	return nil
}

func getVersionForLatestTag(latestTag string, images *model.TagCollection) string {
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

func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

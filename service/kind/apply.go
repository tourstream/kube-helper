package kind

import (
	"fmt"
	"strings"

	"kube-helper/loader"
	"kube-helper/util"

	"kube-helper/model"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *kindService) ApplyKind(kubernetesNamespace string, fileLines []string, namespaceWithoutPrefix string) error {

	fileContent, groupVersionKind, err := k.decoder.Decode([]byte(strings.Join(fileLines, "\n")), nil, nil)

	if err != nil {
		return err
	}
	switch groupVersionKind.Kind {
	case "Secret":
		return k.upsertSecrets(kubernetesNamespace, fileContent.(*coreV1.Secret))
	case "ConfigMap":
		return k.upsertConfigMap(kubernetesNamespace, fileContent.(*coreV1.ConfigMap))
	case "Service":
		return k.upsertService(kubernetesNamespace, fileContent.(*coreV1.Service))
	case "Deployment":
		return k.upsertDeployment(kubernetesNamespace, fileContent.(*apps.Deployment), namespaceWithoutPrefix)
	case "Ingress":
		return k.upsertIngress(kubernetesNamespace, fileContent.(*extensions.Ingress))
	case "CronJob":
		return k.upsertCronJob(kubernetesNamespace, fileContent.(*batch.CronJob), namespaceWithoutPrefix)
	case "PersistentVolume":
		return k.upsertPersistentVolume(fileContent.(*coreV1.PersistentVolume))
	case "PersistentVolumeClaim":
		return k.upsertPersistentVolumeClaim(kubernetesNamespace, fileContent.(*coreV1.PersistentVolumeClaim))
	default:
		return fmt.Errorf("kind %s is not supported", fileContent.GetObjectKind().GroupVersionKind().Kind)
	}
}

func (k *kindService) upsertSecrets(kubernetesNamespace string, secret *coreV1.Secret) error {
	_, err := k.clientSet.CoreV1().Secrets(kubernetesNamespace).Get(secret.Name, metaV1.GetOptions{})

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

func (k *kindService) upsertCronJob(kubernetesNamespace string, cronJob *batch.CronJob, namespaceWithoutPrefix string) error {

	err := k.setImageForContainer(cronJob.Annotations, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, namespaceWithoutPrefix)

	if err != nil {
		return err
	}

	_, err = k.clientSet.BatchV1beta1().CronJobs(kubernetesNamespace).Get(cronJob.Name, metaV1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.BatchV1beta1().CronJobs(kubernetesNamespace).Create(cronJob)

		if err != nil {
			return err
		}

		k.usedKind.cronJob = append(k.usedKind.cronJob, cronJob.Name)

		fmt.Fprintf(writer, "CronJob \"%s\" was generated.\n", cronJob.Name)

		return nil
	}
	_, err = k.clientSet.BatchV1beta1().CronJobs(kubernetesNamespace).Update(cronJob)

	if err != nil {
		return err
	}

	k.usedKind.cronJob = append(k.usedKind.cronJob, cronJob.Name)

	fmt.Fprintf(writer, "CronJob \"%s\" was updated.\n", cronJob.Name)

	return nil
}

func (k *kindService) upsertDeployment(kubernetesNamespace string, deployment *apps.Deployment, namespaceWithoutPrefix string) error {

	err := k.setImageForContainer(deployment.Annotations, deployment.Spec.Template.Spec.Containers, namespaceWithoutPrefix)

	if err != nil {
		return err
	}

	_, err = k.clientSet.AppsV1().Deployments(kubernetesNamespace).Get(deployment.Name, metaV1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.AppsV1().Deployments(kubernetesNamespace).Create(deployment)

		if err != nil {
			return err
		}

		k.usedKind.deployment = append(k.usedKind.deployment, deployment.Name)

		fmt.Fprintf(writer, "Deployment \"%s\" was generated.\n", deployment.Name)

		return nil
	}

	_, err = k.clientSet.AppsV1().Deployments(kubernetesNamespace).Update(deployment)

	if err != nil {
		return err
	}

	k.usedKind.deployment = append(k.usedKind.deployment, deployment.Name)

	fmt.Fprintf(writer, "Deployment \"%s\" was updated.\n", deployment.Name)

	return nil
}

func (k *kindService) upsertService(kubernetesNamespace string, service *coreV1.Service) error {

	existingService, err := k.clientSet.CoreV1().Services(kubernetesNamespace).Get(service.Name, metaV1.GetOptions{})

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
		// TODO add better check which port is which, for now take the same ports like before so that the backend still works with it
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

func (k *kindService) upsertConfigMap(kubernetesNamespace string, configMap *coreV1.ConfigMap) error {

	_, err := k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).Get(configMap.Name, metaV1.GetOptions{})

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

func (k *kindService) upsertPersistentVolume(persistentVolume *coreV1.PersistentVolume) error {

	_, err := k.clientSet.CoreV1().PersistentVolumes().Get(persistentVolume.Name, metaV1.GetOptions{})

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

func (k *kindService) upsertPersistentVolumeClaim(kubernetesNamespace string, persistentVolumeClaim *coreV1.PersistentVolumeClaim) error {

	existingClaim, err := k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).Get(persistentVolumeClaim.Name, metaV1.GetOptions{})

	if err != nil {
		_, err := k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).Create(persistentVolumeClaim)

		if err != nil {
			return err
		}

		k.usedKind.persistentVolumeClaim = append(k.usedKind.persistentVolumeClaim, persistentVolumeClaim.Name)

		fmt.Fprintf(writer, "PersistentVolumeClaim \"%s\" was generated.\n", persistentVolumeClaim.Name)

		return nil
	}

	// TODO check if change and fail in favour of a change in spec
	// override with existing spec because spec is immutable
	persistentVolumeClaim.Spec = existingClaim.Spec

	_, err = k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).Update(persistentVolumeClaim)

	if err != nil {
		return err
	}

	k.usedKind.persistentVolumeClaim = append(k.usedKind.persistentVolumeClaim, persistentVolumeClaim.Name)

	fmt.Fprintf(writer, "PersistentVolumeClaim \"%s\" was updated.\n", persistentVolumeClaim.Name)

	return nil
}

func (k *kindService) upsertIngress(kubernetesNamespace string, ingress *extensions.Ingress) error {

	_, err := k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Get(ingress.Name, metaV1.GetOptions{})

	funcCall := k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Update
	message := "Ingress \"%s\" was updated.\n"

	if err != nil {
		funcCall = k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Create
		message = "Ingress \"%s\" was generated.\n"
	}

	_, err = funcCall(ingress)
	if err != nil {
		return err
	}

	k.usedKind.ingress = append(k.usedKind.ingress, ingress.Name)
	fmt.Fprintf(writer, message, ingress.Name)

	if err != nil {
		return err
	}

	return nil
}

func (k *kindService) setImageForContainer(annotations map[string]string, containers []coreV1.Container, namespaceWithoutPrefix string) error {

	if _, ok := annotations["imageUpdateStrategy"]; !ok {
		return nil
	}

	for idx, container := range containers {

		if strings.Contains(container.Image, "gcr.io") == false {
			continue
		}

		images, err := k.imagesService.List(loader.Cleanup{ImagePath: container.Image})

		if err != nil {
			return err
		}

		switch annotations["imageUpdateStrategy"] {
		case "latest-branching":

			latestTag := loader.StagingEnvironment + "-" + namespaceWithoutPrefix + "-latest"

			if namespaceWithoutPrefix == loader.StagingEnvironment {
				latestTag = "staging-latest"
			}

			if namespaceWithoutPrefix == loader.ProductionEnvironment {
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
		if util.Contains(manifest.Tags, latestTag) {
			for _, tag := range manifest.Tags {
				if tag != latestTag && tag != "latest" {
					return tag
				}
			}
		}
	}

	return ""
}

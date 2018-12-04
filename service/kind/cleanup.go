package kind

import (
	"fmt"
	"strings"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	list, err := k.clientSet.CoreV1().Secrets(kubernetesNamespace).List(metaV1.ListOptions{})

	if err != nil {
		return err
	}

	var names []string

	for _, listEntry := range list.Items {
		if strings.HasPrefix(listEntry.Name, "default-token-") {
			continue
		}
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.secret) {
		err = k.clientSet.CoreV1().Secrets(kubernetesNamespace).Delete(name, &metaV1.DeleteOptions{})
		if err != nil {
			return err
		}
		fmt.Fprintf(writer, "Secret \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupConfigMaps(kubernetesNamespace string) error {
	list, err := k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).List(metaV1.ListOptions{})

	if err != nil {
		return err
	}

	var names []string

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.configMap) {
		err = k.clientSet.CoreV1().ConfigMaps(kubernetesNamespace).Delete(name, &metaV1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "ConfigMap \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupServices(kubernetesNamespace string) error {
	list, err := k.clientSet.CoreV1().Services(kubernetesNamespace).List(metaV1.ListOptions{})

	if err != nil {
		return err
	}

	var names []string

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.service) {
		err = k.clientSet.CoreV1().Services(kubernetesNamespace).Delete(name, &metaV1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Service \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupDeployment(kubernetesNamespace string) error {
	list, err := k.clientSet.AppsV1().Deployments(kubernetesNamespace).List(metaV1.ListOptions{})

	if err != nil {
		return err
	}

	var names []string

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.deployment) {
		err = k.clientSet.AppsV1().Deployments(kubernetesNamespace).Delete(name, &metaV1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Deployment \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupIngresses(kubernetesNamespace string) error {
	list, err := k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).List(metaV1.ListOptions{})

	if err != nil {
		return err
	}

	var names []string

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.ingress) {
		err = k.clientSet.ExtensionsV1beta1().Ingresses(kubernetesNamespace).Delete(name, &metaV1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Ingress \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupCronjobs(kubernetesNamespace string) error {

	list, err := k.clientSet.BatchV1beta1().CronJobs(kubernetesNamespace).List(metaV1.ListOptions{})

	if err != nil {
		return err
	}

	var names []string

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.cronJob) {
		err = k.clientSet.BatchV1beta1().CronJobs(kubernetesNamespace).Delete(name, &metaV1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "CronJob \"%s\" was removed.\n", name)
	}

	return nil
}

func (k *kindService) cleanupPersistentVolumeClaims(kubernetesNamespace string) error {
	list, err := k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).List(metaV1.ListOptions{})

	if err != nil {
		return err
	}

	var names []string

	for _, listEntry := range list.Items {
		names = append(names, listEntry.Name)
	}

	for _, name := range difference(names, k.usedKind.persistentVolumeClaim) {
		err = k.clientSet.CoreV1().PersistentVolumeClaims(kubernetesNamespace).Delete(name, &metaV1.DeleteOptions{})
		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "PersistentVolumeClaim \"%s\" was removed.\n", name)
	}

	return nil
}

func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}

	var ab []string
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

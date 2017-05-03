package app

import (
	"log"
	"os"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"kube-helper/util"
)

func createKind(kubernetesNamespace string) {
	fileContent, _, err := universalDecoder.Decode(getFileContent(tmpSplitFile), nil, nil)
	util.CheckError(err)
	switch fileContent.GetObjectKind().GroupVersionKind().Kind {
	case "Secret":
		createSecrets(kubernetesNamespace, fileContent.(*v1.Secret))
	case "ConfigMap":
		createConfigMap(kubernetesNamespace, fileContent.(*v1.ConfigMap))
	case "Service":
		createService(kubernetesNamespace, fileContent.(*v1.Service))
	case "Deployment":
		createDeployment(kubernetesNamespace, fileContent.(*v1beta1.Deployment))
	case "Ingress":
		createIngress(kubernetesNamespace, fileContent.(*v1beta1.Ingress))
	default:
		log.Panicf("Kind %s is not supported.", fileContent.GetObjectKind().GroupVersionKind().Kind)
	}
	os.Remove(tmpSplitFile)
}

func createDeployment(kubernetesNamespace string, deployment *v1beta1.Deployment) {

	_, err := clientset.Deployments(kubernetesNamespace).Create(deployment)

	util.CheckError(err)

	log.Printf("Deployment \"%s\" was generated\n", deployment.ObjectMeta.Name)
}

func createIngress(kubernetesNamespace string, ingress *v1beta1.Ingress) {

	_, err := clientset.Ingresses(kubernetesNamespace).Create(ingress)

	util.CheckError(err)

	log.Printf("Ingress \"%s\" was generated\n", ingress.ObjectMeta.Name)
}

func createService(kubernetesNamespace string, service *v1.Service) {
	_, err := clientset.Services(kubernetesNamespace).Create(service)

	util.CheckError(err)

	log.Printf("Service \"%s\" was generated\n", service.ObjectMeta.Name)
}

func createSecrets(kubernetesNamespace string, secret *v1.Secret) {
	_, err := clientset.Secrets(kubernetesNamespace).Create(secret)

	util.CheckError(err)

	log.Printf("Secret \"%s\" was generated\n", secret.ObjectMeta.Name)
}

func createConfigMap(kubernetesNamespace string, configMap *v1.ConfigMap) {
	_, err := clientset.ConfigMaps(kubernetesNamespace).Create(configMap)

	util.CheckError(err)

	log.Printf("ConfigMap \"%s\" was generated\n", configMap.ObjectMeta.Name)
}

func updateKind(kubernetesNamespace string) {
	fileContent, _, err := universalDecoder.Decode(getFileContent(tmpSplitFile), nil, nil)
	util.CheckError(err)
	switch fileContent.GetObjectKind().GroupVersionKind().Kind {
	case "Secret":
		updateSecrets(kubernetesNamespace, fileContent.(*v1.Secret))
	case "ConfigMap":
		updateConfigMap(kubernetesNamespace, fileContent.(*v1.ConfigMap))
	case "Service":
		log.Print("Service update is not supported.")
	case "Deployment":
		updateDeployment(kubernetesNamespace, fileContent.(*v1beta1.Deployment))
	case "Ingress":
		log.Print("Ingress update is not supported.")
	default:
		log.Panicf("Kind %s is not supported.", fileContent.GetObjectKind().GroupVersionKind().Kind)
	}
	os.Remove(tmpSplitFile)
}

func updateDeployment(kubernetesNamespace string, deployment *v1beta1.Deployment) {

	_, err := clientset.Deployments(kubernetesNamespace).Update(deployment)

	util.CheckError(err)

	log.Printf("Deployment \"%s\" was updated\n", deployment.ObjectMeta.Name)
}

func updateSecrets(kubernetesNamespace string, secret *v1.Secret) {
	_, err := clientset.Secrets(kubernetesNamespace).Update(secret)

	util.CheckError(err)

	log.Printf("Secret \"%s\" was updated\n", secret.ObjectMeta.Name)
}

func updateConfigMap(kubernetesNamespace string, configMap *v1.ConfigMap) {
	_, err := clientset.ConfigMaps(kubernetesNamespace).Update(configMap)

	util.CheckError(err)

	log.Printf("ConfigMap \"%s\" was updated\n", configMap.ObjectMeta.Name)
}

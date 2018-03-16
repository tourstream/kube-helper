package kind

import (
	"kube-helper/loader"

	"io"
	"os"

	"kube-helper/service/image"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var writer io.Writer = os.Stdout

type KindInterface interface {
	ApplyKind(kubernetesNamespace string, fileLines []string, namespaceWithoutPrefix string) error
	CleanupKind(kubernetesNamespace string) error
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
	imagesService image.ImagesInterface
	config        loader.Config
	usedKind      usedKind
}

// NewKind is the constructor method and returns a service which implements the KindInterface
// the service is used to apply different kubernetes kinds and also do a cleanup depending on the applied ones
func NewKind(client kubernetes.Interface, imagesService image.ImagesInterface, config loader.Config) KindInterface {
	k := new(kindService)
	k.clientSet = client
	k.imagesService = imagesService
	k.config = config
	k.usedKind = usedKind{}
	k.decoder = scheme.Codecs.UniversalDeserializer()

	return k
}

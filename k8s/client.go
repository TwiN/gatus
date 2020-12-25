package k8s

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TwinProduction/gatus/k8stest"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesClientApi is a minimal interface for interacting with Kubernetes
// Created mostly to make mocking the Kubernetes client easier
type KubernetesClientApi interface {
	GetServices(namespace string) ([]v1.Service, error)
}

// KubernetesClient is a working implementation of KubernetesClientApi
type KubernetesClient struct {
	client *kubernetes.Clientset
}

// GetServices returns a list of services for a given namespace
func (k *KubernetesClient) GetServices(namespace string) ([]v1.Service, error) {
	services, err := k.client.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return services.Items, nil
}

// NewKubernetesClient creates a KubernetesClient
func NewKubernetesClient(client *kubernetes.Clientset) *KubernetesClient {
	return &KubernetesClient{
		client: client,
	}
}

// NewClient creates a Kubernetes client for the given ClusterMode
func NewClient(clusterMode ClusterMode) (KubernetesClientApi, error) {
	var kubeConfig *rest.Config
	var err error
	switch clusterMode {
	case ClusterModeIn:
		kubeConfig, err = rest.InClusterConfig()
	case ClusterModeOut:
		kubeConfig, err = getOutClusterConfig()
	case ClusterModeMock:
		return k8stest.GetMockedKubernetesClient(), nil
	default:
		return nil, fmt.Errorf("invalid cluster mode, try '%s' or '%s'", ClusterModeIn, ClusterModeOut)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster config for mode '%s': %s", clusterMode, err.Error())
	}
	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return NewKubernetesClient(client), nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getOutClusterConfig() (*rest.Config, error) {
	var kubeConfig *string
	if home := homeDir(); home != "" {
		kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	return clientcmd.BuildConfigFromFlags("", *kubeConfig)
}

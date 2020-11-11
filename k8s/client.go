package k8s

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient creates a Kubernetes client for the given ClusterMode
func NewClient(clusterMode ClusterMode) (*kubernetes.Clientset, error) {
	var kubeConfig *rest.Config
	var err error
	switch clusterMode {
	case ClusterModeIn:
		kubeConfig, err = getInClusterConfig()
	case ClusterModeOut:
		kubeConfig, err = getOutClusterConfig()
	default:
		return nil, fmt.Errorf("invalid cluster mode, try '%s' or '%s'", ClusterModeIn, ClusterModeOut)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster config for mode '%s': %s", clusterMode, err.Error())
	}
	return kubernetes.NewForConfig(kubeConfig)
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

func getInClusterConfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}

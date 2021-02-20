package k8s

import (
	"k8s.io/api/core/v1"
)

// GetKubernetesServices return a list of Services from the given namespace
func GetKubernetesServices(client KubernetesClientAPI, namespace string) ([]v1.Service, error) {
	return client.GetServices(namespace)
}

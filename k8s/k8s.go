package k8s

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetKubernetesServices return List of Services from given namespace
func GetKubernetesServices(client *kubernetes.Clientset, ns string) ([]corev1.Service, error) {
	services, err := client.CoreV1().Services(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return services.Items, nil
}

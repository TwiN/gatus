package k8s

import (
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//GetServices return List of Services from given namespace
func GetServices(client *kubernetes.Clientset, ns string) []corev1.Service {
	options := metav1.ListOptions{}
	svcs, err := client.CoreV1().Services(ns).List(options)
	if err != nil {
		log.Printf("[Discovery] : Error getting Services Err: %v", err)
		return []corev1.Service{}
	}
	return svcs.Items
}

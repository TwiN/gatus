package k8stest

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	mockedKubernetesClient *MockKubernetesClient
)

// MockKubernetesClient is a mocked implementation of k8s.KubernetesClientApi
type MockKubernetesClient struct {
	Services []v1.Service
}

// GetServices returns a list of services in a given namespace
func (mock *MockKubernetesClient) GetServices(namespace string) ([]v1.Service, error) {
	var services []v1.Service
	for _, service := range mock.Services {
		if service.Namespace == namespace {
			services = append(services, service)
		}
	}
	return services, nil
}

// GetMockedKubernetesClient returns a mocked implementation of k8s.KubernetesClientApi
func GetMockedKubernetesClient() *MockKubernetesClient {
	if mockedKubernetesClient != nil {
		return mockedKubernetesClient
	}
	InitializeMockedKubernetesClient(nil)
	return mockedKubernetesClient
}

// InitializeMockedKubernetesClient initializes a MockKubernetesClient with a given list of services
func InitializeMockedKubernetesClient(services []v1.Service) {
	mockedKubernetesClient = &MockKubernetesClient{
		Services: services,
	}
}

// CreateTestServices creates a mocked service for testing purposes
func CreateTestServices(name, namespace string) v1.Service {
	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{},
	}
}

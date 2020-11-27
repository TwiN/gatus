package core

import "testing"

func TestNewServiceStatus(t *testing.T) {
	service := &Service{Group: "test"}
	serviceStatus := NewServiceStatus(service)
	if serviceStatus.Group != service.Group {
		t.Errorf("expected %s, got %s", service.Group, serviceStatus.Group)
	}
}

func TestServiceStatus_AddResult(t *testing.T) {
	service := &Service{Group: "test"}
	serviceStatus := NewServiceStatus(service)
	for i := 0; i < 50; i++ {
		serviceStatus.AddResult(&Result{})
	}
	if len(serviceStatus.Results) != 20 {
		t.Errorf("expected serviceStatus.Results to not exceed a length of 20")
	}
}

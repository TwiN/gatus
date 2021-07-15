package core

import (
	"testing"
)

func TestNewServiceStatus(t *testing.T) {
	service := &Service{Name: "name", Group: "group"}
	serviceStatus := NewServiceStatus(service.Key(), service.Group, service.Name)
	if serviceStatus.Name != service.Name {
		t.Errorf("expected %s, got %s", service.Name, serviceStatus.Name)
	}
	if serviceStatus.Group != service.Group {
		t.Errorf("expected %s, got %s", service.Group, serviceStatus.Group)
	}
	if serviceStatus.Key != "group_name" {
		t.Errorf("expected %s, got %s", "group_name", serviceStatus.Key)
	}
}

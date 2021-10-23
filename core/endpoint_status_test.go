package core

import (
	"testing"
)

func TestNewEndpointStatus(t *testing.T) {
	endpoint := &Endpoint{Name: "name", Group: "group"}
	status := NewEndpointStatus(endpoint.Group, endpoint.Name)
	if status.Name != endpoint.Name {
		t.Errorf("expected %s, got %s", endpoint.Name, status.Name)
	}
	if status.Group != endpoint.Group {
		t.Errorf("expected %s, got %s", endpoint.Group, status.Group)
	}
	if status.Key != "group_name" {
		t.Errorf("expected %s, got %s", "group_name", status.Key)
	}
}

package endpoint

import (
	"testing"
)

func TestNewEndpointStatus(t *testing.T) {
	ep := &Endpoint{Name: "name", Group: "group"}
	status := NewStatus(ep.Group, ep.Name)
	if status.Name != ep.Name {
		t.Errorf("expected %s, got %s", ep.Name, status.Name)
	}
	if status.Group != ep.Group {
		t.Errorf("expected %s, got %s", ep.Group, status.Group)
	}
	if status.Key != "group_name" {
		t.Errorf("expected %s, got %s", "group_name", status.Key)
	}
}

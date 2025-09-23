package endpoint

import (
	"testing"
)

func TestNewEndpointStatus(t *testing.T) {
	ep := &Endpoint{Name: "name", Groups: []string{"group"}}
	status := NewStatus(ep.Groups, ep.Name)
	if status.Name != ep.Name {
		t.Errorf("expected %s, got %s", ep.Name, status.Name)
	}
	if status.Groups[0] != ep.Groups[0] {
		t.Errorf("expected %s, got %s", ep.Groups[0], status.Groups[0])
	}
	if status.Key != "group_name" {
		t.Errorf("expected %s, got %s", "group_name", status.Key)
	}
}

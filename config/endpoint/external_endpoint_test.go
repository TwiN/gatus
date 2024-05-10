package endpoint

import (
	"testing"
)

func TestExternalEndpoint_ToEndpoint(t *testing.T) {
	externalEndpoint := &ExternalEndpoint{
		Name:  "name",
		Group: "group",
	}
	convertedEndpoint := externalEndpoint.ToEndpoint()
	if externalEndpoint.Name != convertedEndpoint.Name {
		t.Errorf("expected %s, got %s", externalEndpoint.Name, convertedEndpoint.Name)
	}
	if externalEndpoint.Group != convertedEndpoint.Group {
		t.Errorf("expected %s, got %s", externalEndpoint.Group, convertedEndpoint.Group)
	}
	if externalEndpoint.Key() != convertedEndpoint.Key() {
		t.Errorf("expected %s, got %s", externalEndpoint.Key(), convertedEndpoint.Key())
	}
	if externalEndpoint.DisplayName() != convertedEndpoint.DisplayName() {
		t.Errorf("expected %s, got %s", externalEndpoint.DisplayName(), convertedEndpoint.DisplayName())
	}
}

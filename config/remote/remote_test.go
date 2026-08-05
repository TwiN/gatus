package remote

import "testing"

func TestPrefixedKeyAndParsePrefixedKey(t *testing.T) {
	t.Parallel()

	key := PrefixedKey(1, "core_backend")
	if key != "@remote:1:core_backend" {
		t.Fatalf("expected @remote:1:core_backend, got %s", key)
	}
	if !IsPrefixedKey(key) {
		t.Fatal("expected key to be prefixed")
	}

	instanceIndex, originalKey, ok := ParsePrefixedKey(key)
	if !ok {
		t.Fatal("expected key to parse")
	}
	if instanceIndex != 1 || originalKey != "core_backend" {
		t.Fatalf("expected 1/core_backend, got %d/%s", instanceIndex, originalKey)
	}

	if _, _, ok := ParsePrefixedKey("core_backend"); ok {
		t.Fatal("expected local key not to parse as remote key")
	}
	if _, _, ok := ParsePrefixedKey("@remote/"); ok {
		t.Fatal("expected invalid remote key not to parse")
	}
}

func TestInstanceEndpointBaseURL(t *testing.T) {
	t.Parallel()

	instance := Instance{URL: "https://status.example.org/api/v1/endpoints/statuses"}
	if instance.EndpointBaseURL() != "https://status.example.org/api/v1/endpoints" {
		t.Fatalf("unexpected base URL: %s", instance.EndpointBaseURL())
	}
}

func TestInstanceBuildEndpointURL(t *testing.T) {
	t.Parallel()

	instance := Instance{URL: "https://status.example.org/api/v1/endpoints/statuses"}
	got := instance.BuildEndpointURL("core_backend", "/statuses")
	want := "https://status.example.org/api/v1/endpoints/core_backend/statuses"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

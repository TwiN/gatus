package remote

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/client"
)

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

func TestValidateAndSetDefaultsRejectsInvalidURLScheme(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Instances: []Instance{{
			URL: "file:///tmp/statuses",
		}},
	}
	if err := cfg.ValidateAndSetDefaults(); err == nil {
		t.Fatal("expected validation error for file:// scheme")
	}
}

func TestValidateAndSetDefaultsRejectsPrivateNetworkWithoutOptIn(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Instances: []Instance{{
			URL: "http://127.0.0.1:8080/api/v1/endpoints/statuses",
		}},
	}
	if err := cfg.ValidateAndSetDefaults(); err == nil {
		t.Fatal("expected validation error for private network URL")
	}
}

func TestValidateAndSetDefaultsAllowsPrivateNetworkWithOptIn(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Instances: []Instance{{
			URL:                  "http://127.0.0.1:8080/api/v1/endpoints/statuses",
			AllowPrivateNetworks: true,
		}},
	}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("expected validation to pass, got %s", err.Error())
	}
	if cfg.Instances[0].EndpointBaseURL() != "http://127.0.0.1:8080/api/v1/endpoints" {
		t.Fatalf("unexpected cached base URL: %s", cfg.Instances[0].EndpointBaseURL())
	}
}

func TestValidateAndSetDefaultsRejectsEmptyURL(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Instances: []Instance{{
			URL: "   ",
		}},
	}
	if err := cfg.ValidateAndSetDefaults(); err == nil {
		t.Fatal("expected validation error for empty URL")
	}
}

func TestApplyRequestHeadersUsesInstanceAuthorizationExclusively(t *testing.T) {
	t.Parallel()

	request, err := httpNewRequest("https://status.example.org/api/v1/endpoints/statuses")
	if err != nil {
		t.Fatalf("failed to create request: %s", err.Error())
	}

	instance := Instance{Authorization: "Bearer configured"}
	forwardHeaders := make(map[string][]string)
	forwardHeaders["Authorization"] = []string{"Bearer inbound"}
	forwardHeaders["Cookie"] = []string{"session=abc"}

	instance.ApplyRequestHeaders(request, http.Header(forwardHeaders))
	if request.Header.Get("Authorization") != "Bearer configured" {
		t.Fatalf("expected configured authorization, got %q", request.Header.Get("Authorization"))
	}
	if request.Header.Get("Cookie") != "" {
		t.Fatal("expected cookie not to be forwarded when instance authorization is configured")
	}
}

func httpNewRequest(url string) (*http.Request, error) {
	return http.NewRequest(http.MethodGet, url, http.NoBody)
}

func TestValidateAndSetDefaultsRejectsURLWithoutStatusesPath(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Instances: []Instance{{
			URL: "https://status.example.org/",
		}},
	}
	if err := cfg.ValidateAndSetDefaults(); err == nil {
		t.Fatal("expected validation error for URL without /statuses path")
	}
}

func TestMaxResponseBodySizeDefaultsToTenMiB(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	if cfg.MaxResponseBodySize() != defaultMaxResponseBody {
		t.Fatalf("expected default max response body of %d, got %d", defaultMaxResponseBody, cfg.MaxResponseBodySize())
	}
}

func TestMaxResponseBodySizeUsesConfiguredValue(t *testing.T) {
	t.Parallel()

	cfg := &Config{MaxResponseBody: 2048}
	if cfg.MaxResponseBodySize() != 2048 {
		t.Fatalf("expected max response body of 2048, got %d", cfg.MaxResponseBodySize())
	}
}

func TestValidateAndSetDefaultsUsesSecureTLSDefaults(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Instances: []Instance{{
			URL: "https://status.example.org/api/v1/endpoints/statuses",
		}},
	}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("expected validation to pass, got %s", err.Error())
	}
	if cfg.ClientConfig.Insecure {
		t.Fatal("expected remote client TLS verification to be enabled by default")
	}
	if cfg.ClientConfig.Timeout != 10*time.Second {
		t.Fatalf("expected default client timeout of 10s, got %s", cfg.ClientConfig.Timeout)
	}
}

func TestHostResolvesToPrivateNetworkRejectsPrivateIP(t *testing.T) {
	t.Parallel()

	err := hostResolvesToPrivateNetwork("internal.example.org", func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.0.0.1")}, nil
	})
	if err == nil {
		t.Fatal("expected error when hostname resolves to a private IP")
	}
}

func TestHostResolvesToPrivateNetworkAllowsLookupFailure(t *testing.T) {
	t.Parallel()

	err := hostResolvesToPrivateNetwork("missing.example.org", func(host string) ([]net.IP, error) {
		return nil, &net.DNSError{IsNotFound: true, Name: host}
	})
	if err != nil {
		t.Fatalf("expected lookup failure to be tolerated, got %s", err.Error())
	}
}

func TestCircuitBreakerConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg := &CircuitBreakerConfig{}
	if cfg.FailureThresholdOrDefault() != defaultCircuitBreakerFailureThreshold {
		t.Fatalf("expected default failure threshold of %d, got %d", defaultCircuitBreakerFailureThreshold, cfg.FailureThresholdOrDefault())
	}
	if cfg.OpenDurationOrDefault() != defaultCircuitBreakerOpenDuration {
		t.Fatalf("expected default open duration of %s, got %s", defaultCircuitBreakerOpenDuration, cfg.OpenDurationOrDefault())
	}
}

func TestValidateAndSetDefaultsPreservesExplicitClientConfig(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Instances: []Instance{{
			URL: "https://status.example.org/api/v1/endpoints/statuses",
		}},
		ClientConfig: &client.Config{
			Timeout:  3 * time.Second,
			Insecure: true,
		},
	}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("expected validation to pass, got %s", err.Error())
	}
	if !cfg.ClientConfig.Insecure {
		t.Fatal("expected explicit insecure client setting to be preserved")
	}
	if cfg.ClientConfig.Timeout != 3*time.Second {
		t.Fatalf("expected explicit client timeout of 3s, got %s", cfg.ClientConfig.Timeout)
	}
}

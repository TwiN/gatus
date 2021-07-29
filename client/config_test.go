package client

import (
	"net/http"
	"testing"
	"time"
)

func TestConfig_getHTTPClient(t *testing.T) {
	insecureConfig := &Config{Insecure: true}
	insecureConfig.ValidateAndSetDefaults()
	insecureClient := insecureConfig.getHTTPClient()
	if !(insecureClient.Transport).(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Error("expected Config.Insecure set to true to cause the HTTP client to skip certificate verification")
	}
	if insecureClient.Timeout != defaultHTTPTimeout {
		t.Error("expected Config.Timeout to default the HTTP client to a timeout of 10s")
	}
	request, _ := http.NewRequest("GET", "", nil)
	if err := insecureClient.CheckRedirect(request, nil); err != nil {
		t.Error("expected Config.IgnoreRedirect set to false to cause the HTTP client's CheckRedirect to return nil")
	}

	secureConfig := &Config{IgnoreRedirect: true, Timeout: 5 * time.Second}
	secureConfig.ValidateAndSetDefaults()
	secureClient := secureConfig.getHTTPClient()
	if (secureClient.Transport).(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Error("expected Config.Insecure set to false to cause the HTTP client to not skip certificate verification")
	}
	if secureClient.Timeout != 5*time.Second {
		t.Error("expected Config.Timeout to cause the HTTP client to have a timeout of 5s")
	}
	request, _ = http.NewRequest("GET", "", nil)
	if err := secureClient.CheckRedirect(request, nil); err != http.ErrUseLastResponse {
		t.Error("expected Config.IgnoreRedirect set to true to cause the HTTP client's CheckRedirect to return http.ErrUseLastResponse")
	}
}

package client

import (
	"testing"
)

func TestGetHttpClient(t *testing.T) {
	if secureHTTPClient != nil {
		t.Error("secureHTTPClient should've been nil since it hasn't been called a single time yet")
	}
	if insecureHTTPClient != nil {
		t.Error("insecureHTTPClient should've been nil since it hasn't been called a single time yet")
	}
	_ = GetHTTPClient(false)
	if secureHTTPClient == nil {
		t.Error("secureHTTPClient shouldn't have been nil, since it has been called once")
	}
	if insecureHTTPClient != nil {
		t.Error("insecureHTTPClient should've been nil since it hasn't been called a single time yet")
	}
	_ = GetHTTPClient(true)
	if secureHTTPClient == nil {
		t.Error("secureHTTPClient shouldn't have been nil, since it has been called once")
	}
	if insecureHTTPClient == nil {
		t.Error("insecureHTTPClient shouldn't have been nil, since it has been called once")
	}
}

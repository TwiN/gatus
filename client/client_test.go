package client

import "testing"

func TestGetHttpClient(t *testing.T) {
	if secureHttpClient != nil {
		t.Error("secureHttpClient should've been nil since it hasn't been called a single time yet")
	}
	if insecureHttpClient != nil {
		t.Error("insecureHttpClient should've been nil since it hasn't been called a single time yet")
	}
	_ = GetHttpClient(false)
	if secureHttpClient == nil {
		t.Error("secureHttpClient shouldn't have been nil, since it has been called once")
	}
	if insecureHttpClient != nil {
		t.Error("insecureHttpClient should've been nil since it hasn't been called a single time yet")
	}
	_ = GetHttpClient(true)
	if secureHttpClient == nil {
		t.Error("secureHttpClient shouldn't have been nil, since it has been called once")
	}
	if insecureHttpClient == nil {
		t.Error("insecureHttpClient shouldn't have been nil, since it has been called once")
	}
}

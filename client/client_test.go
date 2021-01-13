package client

import (
	"testing"
	"time"
)

func TestGetHTTPClient(t *testing.T) {
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

func TestPing(t *testing.T) {
	pingTimeout = 500 * time.Millisecond
	if success, rtt := Ping("127.0.0.1"); !success {
		t.Error("expected true")
		if rtt == 0 {
			t.Error("Round-trip time returned on success should've higher than 0")
		}
	}
	if success, rtt := Ping("256.256.256.256"); success {
		t.Error("expected false, because the IP is invalid")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	if success, rtt := Ping("192.168.152.153"); success {
		t.Error("expected false, because the IP is valid but the host should be unreachable")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
}

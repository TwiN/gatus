package client

import "testing"

func TestGetHttpClient(t *testing.T) {
	if client != nil {
		t.Error("client should've been nil since it hasn't been called a single time yet")
	}
	_ = GetHttpClient()
	if client == nil {
		t.Error("client shouldn't have been nil, since it has been called once")
	}
}

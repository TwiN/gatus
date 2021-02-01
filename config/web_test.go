package config

import (
	"testing"
)

func TestWebConfig_SocketAddress(t *testing.T) {
	web := &WebConfig{
		Address: "0.0.0.0",
		Port:    8081,
	}
	if web.SocketAddress() != "0.0.0.0:8081" {
		t.Errorf("expected %s, got %s", "0.0.0.0:8081", web.SocketAddress())
	}
}

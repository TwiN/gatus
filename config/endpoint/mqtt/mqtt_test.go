package mqtt

import (
	"errors"
	"testing"
)

func TestMQTT_validate(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Error("expected an error")
	} else if !errors.Is(err, ErrEndpointWithoutMQTTTopic) {
		t.Errorf("expected error to be '%v', got '%v'", ErrEndpointWithoutMQTTTopic, err)
	}
	cfg.Username = "username"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got '%v'", err)
	}
	cfg.Password = "password"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got '%v'", err)
	}
}

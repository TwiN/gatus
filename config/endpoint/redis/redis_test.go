package redis

import (
	"errors"
	"testing"
)

func TestRedis_Validate(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Error("expected an error when password is missing")
	} else if !errors.Is(err, ErrEndpointWithoutRedisPassword) {
		t.Errorf("expected error to be '%v', got '%v'", ErrEndpointWithoutRedisPassword, err)
	}

	cfg.Password = "password"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got '%v'", err)
	}
}

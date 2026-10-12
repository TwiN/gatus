package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TwiN/gatus/v5/config"
)

// TestTryLoadNewConfig_SkipInvalidConfigUpdate is a regression test for the fix
// that moves stop(cfg) to after a successful load, ensuring the running config is
// not torn down when the incoming configuration file is invalid.
//
// It verifies two things:
//  1. When SkipInvalidConfigUpdate is true and loadFn returns an error,
//     tryLoadNewConfig returns (nil, nil) — the nil cfg signals to the caller
//     that it must NOT stop the old config (no stop-before-load).
//  2. The old cfg's endpoints are preserved — the running config keeps working.
func TestTryLoadNewConfig_SkipInvalidConfigUpdate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configFilePath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(configFilePath, []byte(`
skip-invalid-config-update: true
endpoints:
  - name: website
    url: https://twin.sh/health
    conditions:
      - "[STATUS] == 200"
`), 0o644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	cfg, err := config.LoadConfiguration(configFilePath)
	if err != nil {
		t.Fatalf("failed to load initial configuration: %v", err)
	}
	if len(cfg.Endpoints) == 0 {
		t.Fatal("expected at least one endpoint in the initial configuration")
	}
	endpointName := cfg.Endpoints[0].Name

	// A loadFn that always returns an error (simulating a config that fails validation).
	alwaysFail := func() (*config.Config, error) {
		return nil, errors.New("configuration should contain at least one endpoint or suite")
	}

	gotCfg, gotErr := tryLoadNewConfig(cfg, alwaysFail)

	// 1. tryLoadNewConfig must signal "skip — do not stop the running config":
	//    nil cfg and nil error tell the caller to continue without calling stop().
	if gotErr != nil {
		t.Errorf("expected nil error (skip path), got: %v", gotErr)
	}
	if gotCfg != nil {
		t.Errorf("expected nil cfg (skip path, caller must not stop old config), got non-nil cfg")
	}

	// 2. The old config's endpoints must be untouched — the running config keeps working.
	if len(cfg.Endpoints) == 0 {
		t.Fatal("expected cfg.Endpoints to remain non-empty; the running config must not be cleared on an invalid update")
	}
	if cfg.Endpoints[0].Name != endpointName {
		t.Errorf("expected endpoint name %q to be preserved after skipping invalid update, got %q", endpointName, cfg.Endpoints[0].Name)
	}
}

// TestTryLoadNewConfig_NoSkip verifies that when SkipInvalidConfigUpdate is false,
// tryLoadNewConfig propagates the error so the caller can panic (default behavior).
func TestTryLoadNewConfig_NoSkip(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{SkipInvalidConfigUpdate: false}
	sentinel := errors.New("sentinel error")
	alwaysFail := func() (*config.Config, error) { return nil, sentinel }

	gotCfg, gotErr := tryLoadNewConfig(cfg, alwaysFail)
	if !errors.Is(gotErr, sentinel) {
		t.Errorf("expected sentinel error to be propagated, got: %v", gotErr)
	}
	if gotCfg != nil {
		t.Errorf("expected nil cfg on error, got non-nil")
	}
}

// TestTryLoadNewConfig_ValidConfig verifies the happy path: a successful load
// returns the new config and no error.
func TestTryLoadNewConfig_ValidConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configFilePath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configFilePath, []byte(`
endpoints:
  - name: website
    url: https://twin.sh/health
    conditions:
      - "[STATUS] == 200"
`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	cfg := &config.Config{SkipInvalidConfigUpdate: true}
	loadFn := func() (*config.Config, error) {
		return config.LoadConfiguration(configFilePath)
	}

	gotCfg, gotErr := tryLoadNewConfig(cfg, loadFn)
	if gotErr != nil {
		t.Fatalf("expected no error on valid config, got: %v", gotErr)
	}
	if gotCfg == nil {
		t.Fatal("expected non-nil cfg on successful load")
	}
}

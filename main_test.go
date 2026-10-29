package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TwiN/gatus/v5/config"
)

// TestTryLoadNewConfig_SkipInvalidConfigUpdate is a regression test for the fix
// that moves the stop call to after a successful load, ensuring the running config
// is not torn down when the incoming configuration file is invalid.
//
// It verifies three things:
//  1. When SkipInvalidConfigUpdate is true and loadFn returns an error,
//     tryLoadNewConfig returns (nil, nil).
//  2. stopFn is NOT called — the old config must keep running.
//  3. The old cfg's endpoints are preserved.
//
// Revert check: temporarily calling stopFn(cfg) before loadFn() in
// tryLoadNewConfig (as master's listenToConfigurationFileChanges did) causes
// assertion (2) to fail with "expected stopFn not to be called …".
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

	// Spy: records whether the stop function was invoked.
	stopped := false
	spyStop := func(_ *config.Config) { stopped = true }

	// A loadFn that always fails (simulating a config that fails validation).
	alwaysFail := func() (*config.Config, error) {
		return nil, errors.New("configuration should contain at least one endpoint or suite")
	}

	gotCfg, gotErr := tryLoadNewConfig(cfg, spyStop, alwaysFail)

	// 1. Must signal "skip — do not replace the old config".
	if gotErr != nil {
		t.Errorf("expected nil error (skip path), got: %v", gotErr)
	}
	if gotCfg != nil {
		t.Errorf("expected nil cfg (skip path), got non-nil cfg")
	}

	// 2. stopFn must NOT have been called — the old config keeps running.
	if stopped {
		t.Error("expected stopFn not to be called when the new config is invalid and SkipInvalidConfigUpdate is true, but it was called")
	}

	// 3. The old config's endpoints must be untouched.
	if len(cfg.Endpoints) == 0 {
		t.Fatal("expected cfg.Endpoints to remain non-empty; the running config must not be cleared on an invalid update")
	}
	if cfg.Endpoints[0].Name != endpointName {
		t.Errorf("expected endpoint name %q to be preserved after skipping invalid update, got %q", endpointName, cfg.Endpoints[0].Name)
	}
}

// TestTryLoadNewConfig_NoSkip verifies that when SkipInvalidConfigUpdate is false,
// tryLoadNewConfig propagates the error so the caller can panic (default behavior).
// stopFn must also not be called on failure.
func TestTryLoadNewConfig_NoSkip(t *testing.T) {
	t.Parallel()
	stopped := false
	spyStop := func(_ *config.Config) { stopped = true }

	cfg := &config.Config{SkipInvalidConfigUpdate: false}
	sentinel := errors.New("sentinel error")
	alwaysFail := func() (*config.Config, error) { return nil, sentinel }

	gotCfg, gotErr := tryLoadNewConfig(cfg, spyStop, alwaysFail)
	if !errors.Is(gotErr, sentinel) {
		t.Errorf("expected sentinel error to be propagated, got: %v", gotErr)
	}
	if gotCfg != nil {
		t.Errorf("expected nil cfg on error, got non-nil")
	}
	if stopped {
		t.Error("expected stopFn not to be called on load failure, but it was called")
	}
}

// TestTryLoadNewConfig_ValidConfig verifies the happy path: a successful load
// returns the new config and stopFn is called exactly once.
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
	stopCount := 0
	spyStop := func(_ *config.Config) { stopCount++ }

	cfg := &config.Config{SkipInvalidConfigUpdate: true}
	loadFn := func() (*config.Config, error) {
		return config.LoadConfiguration(configFilePath)
	}

	gotCfg, gotErr := tryLoadNewConfig(cfg, spyStop, loadFn)
	if gotErr != nil {
		t.Fatalf("expected no error on valid config, got: %v", gotErr)
	}
	if gotCfg == nil {
		t.Fatal("expected non-nil cfg on successful load")
	}
	if stopCount != 1 {
		t.Errorf("expected stopFn to be called exactly once on successful load, got %d calls", stopCount)
	}
}

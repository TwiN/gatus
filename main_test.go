package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
)

// TestInitializeStorageSeedsLastReminderSentOnRestore covers the three restore
// branches in initializeStorage (regular, external, suite). LastReminderSent
// is yaml:"-" and is never persisted, so after a restart it is zero even when
// Triggered is restored from storage. Seeding it to time.Now() is what keeps
// the first post-restart reminder from firing immediately (#1854).
func TestInitializeStorageSeedsLastReminderSentOnRestore(t *testing.T) {
	enabled := true
	newAlert := func() *alert.Alert {
		return &alert.Alert{
			Type:                    alert.TypeCustom,
			Enabled:                 &enabled,
			FailureThreshold:        2,
			SuccessThreshold:        3,
			Triggered:               true,
			MinimumReminderInterval: time.Hour,
		}
	}

	persistedRegular := &endpoint.Endpoint{
		Name:                   "regular",
		URL:                    "https://example.com",
		Alerts:                 []*alert.Alert{newAlert()},
		NumberOfFailuresInARow: 2,
	}
	persistedExternal := &endpoint.ExternalEndpoint{
		Name:                   "external",
		Group:                  "g",
		Token:                  "token",
		Alerts:                 []*alert.Alert{newAlert()},
		NumberOfFailuresInARow: 2,
	}
	persistedSuiteEndpoint := &endpoint.Endpoint{
		Name:                   "suite-ep",
		URL:                    "https://suite.example",
		Alerts:                 []*alert.Alert{newAlert()},
		NumberOfFailuresInARow: 2,
	}

	storageCfg := &storage.Config{Type: storage.TypeSQLite, Path: filepath.Join(t.TempDir(), "gatus.db")}
	if err := storageCfg.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(storageCfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Get().UpsertTriggeredEndpointAlert(persistedRegular, persistedRegular.Alerts[0]); err != nil {
		t.Fatal(err)
	}
	if err := store.Get().UpsertTriggeredEndpointAlert(persistedExternal.ToEndpoint(), persistedExternal.Alerts[0]); err != nil {
		t.Fatal(err)
	}
	if err := store.Get().UpsertTriggeredEndpointAlert(persistedSuiteEndpoint, persistedSuiteEndpoint.Alerts[0]); err != nil {
		t.Fatal(err)
	}
	store.Get().Close()

	// Fresh objects as after YAML load: Triggered=false, LastReminderSent zero.
	regular := &endpoint.Endpoint{
		Name:   "regular",
		URL:    "https://example.com",
		Alerts: []*alert.Alert{newAlert()},
	}
	regular.Alerts[0].Triggered = false
	ee := &endpoint.ExternalEndpoint{
		Name:   "external",
		Group:  "g",
		Token:  "token",
		Alerts: []*alert.Alert{newAlert()},
	}
	ee.Alerts[0].Triggered = false
	sep := &endpoint.Endpoint{
		Name:   "suite-ep",
		URL:    "https://suite.example",
		Alerts: []*alert.Alert{newAlert()},
	}
	sep.Alerts[0].Triggered = false
	s := &suite.Suite{Name: "suite", Endpoints: []*endpoint.Endpoint{sep}}

	cfg := &config.Config{
		Storage:           storageCfg,
		Endpoints:         []*endpoint.Endpoint{regular},
		ExternalEndpoints: []*endpoint.ExternalEndpoint{ee},
		Suites:            []*suite.Suite{s},
	}
	initializeStorage(cfg)
	t.Cleanup(func() { store.Get().Close() })

	tests := []struct {
		name             string
		triggered        bool
		lastReminderSent time.Time
	}{
		{name: "regular-endpoint", triggered: regular.Alerts[0].Triggered, lastReminderSent: regular.LastReminderSent},
		{name: "external-endpoint", triggered: ee.Alerts[0].Triggered, lastReminderSent: ee.LastReminderSent},
		{name: "suite-endpoint", triggered: sep.Alerts[0].Triggered, lastReminderSent: sep.LastReminderSent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.triggered {
				t.Fatalf("%s: expected Triggered to be restored from storage", tt.name)
			}
			if tt.lastReminderSent.IsZero() {
				t.Fatalf("%s: expected LastReminderSent to be seeded on restore", tt.name)
			}
			if time.Since(tt.lastReminderSent) > time.Minute {
				t.Fatalf("%s: expected LastReminderSent to be seeded to approximately now, got %s", tt.name, tt.lastReminderSent)
			}
		})
	}
}

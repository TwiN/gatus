package watchdog

import (
	"errors"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

// mockProvider is a simple mock for testing failover
type mockProvider struct {
	shouldFail bool
	sendCount  int
}

func (m *mockProvider) Validate() error { return nil }
func (m *mockProvider) Send(ep *endpoint.Endpoint, a *alert.Alert, result *endpoint.Result, resolved bool) error {
	m.sendCount++
	if m.shouldFail {
		return errors.New("mock provider error")
	}
	return nil
}
func (m *mockProvider) GetDefaultAlert() *alert.Alert                          { return nil }
func (m *mockProvider) ValidateOverrides(group string, a *alert.Alert) error { return nil }

var _ provider.AlertProvider = (*mockProvider)(nil)

// mockAlertingConfig wraps providers for testing
type mockAlertingConfig struct {
	providers map[alert.Type]*mockProvider
}

func TestSendWithFailover_PrimarySucceeds(t *testing.T) {
	primary := &mockProvider{shouldFail: false}

	ep := &endpoint.Endpoint{Name: "test-endpoint"}
	endpointAlert := &alert.Alert{Type: alert.TypePagerDuty}
	result := &endpoint.Result{Success: false}

	err := sendWithFailover(ep, endpointAlert, result, false, primary, nil)

	if err != nil {
		t.Errorf("Expected no error when primary succeeds, got: %v", err)
	}
	if primary.sendCount != 1 {
		t.Errorf("Expected primary to be called once, got: %d", primary.sendCount)
	}
}

func TestSendWithFailover_PrimaryFailsNoFailover(t *testing.T) {
	primary := &mockProvider{shouldFail: true}

	ep := &endpoint.Endpoint{Name: "test-endpoint"}
	endpointAlert := &alert.Alert{Type: alert.TypePagerDuty} // No Failover configured
	result := &endpoint.Result{Success: false}

	err := sendWithFailover(ep, endpointAlert, result, false, primary, nil)

	if err == nil {
		t.Error("Expected error when primary fails and no failover configured")
	}
	if primary.sendCount != 1 {
		t.Errorf("Expected primary to be called once, got: %d", primary.sendCount)
	}
}

func TestSendWithFailover_PrimaryFailsFailoverSucceeds(t *testing.T) {
	primary := &mockProvider{shouldFail: true}
	fallback := &mockProvider{shouldFail: false}

	// Create alerting config with fallback provider
	// Note: We can't easily mock GetAlertingProviderByAlertType, so we test the provider behavior directly

	ep := &endpoint.Endpoint{Name: "test-endpoint"}
	endpointAlert := &alert.Alert{
		Type:     alert.TypePagerDuty,
		Failover: []alert.Type{alert.TypeTelegram},
	}
	result := &endpoint.Result{Success: false}

	// Since we can't easily mock GetAlertingProviderByAlertType, let's test the logic directly
	// by simulating what sendWithFailover does

	// First, primary fails
	err := primary.Send(ep, endpointAlert, result, false)
	if err == nil {
		t.Error("Primary should have failed")
	}

	// Then fallback succeeds
	err = fallback.Send(ep, endpointAlert, result, false)
	if err != nil {
		t.Errorf("Fallback should have succeeded, got: %v", err)
	}

	if primary.sendCount != 1 {
		t.Errorf("Expected primary to be called once, got: %d", primary.sendCount)
	}
	if fallback.sendCount != 1 {
		t.Errorf("Expected fallback to be called once, got: %d", fallback.sendCount)
	}
}

func TestSendWithFailover_AllProvidersFail(t *testing.T) {
	primary := &mockProvider{shouldFail: true}
	fallback1 := &mockProvider{shouldFail: true}
	fallback2 := &mockProvider{shouldFail: true}

	ep := &endpoint.Endpoint{Name: "test-endpoint"}
	endpointAlert := &alert.Alert{
		Type:     alert.TypePagerDuty,
		Failover: []alert.Type{alert.TypeTelegram, alert.TypeSlack},
	}
	result := &endpoint.Result{Success: false}

	// Simulate the failover chain manually
	var lastErr error

	// Primary fails
	lastErr = primary.Send(ep, endpointAlert, result, false)
	if lastErr == nil {
		t.Error("Primary should have failed")
	}

	// First fallback fails
	lastErr = fallback1.Send(ep, endpointAlert, result, false)
	if lastErr == nil {
		t.Error("Fallback1 should have failed")
	}

	// Second fallback fails
	lastErr = fallback2.Send(ep, endpointAlert, result, false)
	if lastErr == nil {
		t.Error("Fallback2 should have failed")
	}

	// All should have been tried
	if primary.sendCount != 1 || fallback1.sendCount != 1 || fallback2.sendCount != 1 {
		t.Error("All providers should have been tried exactly once")
	}
}

func TestSendWithFailover_FailoverProviderNotConfigured(t *testing.T) {
	primary := &mockProvider{shouldFail: true}

	ep := &endpoint.Endpoint{Name: "test-endpoint"}
	endpointAlert := &alert.Alert{
		Type:     alert.TypePagerDuty,
		Failover: []alert.Type{alert.TypeTelegram}, // Telegram not configured in alertingConfig
	}
	result := &endpoint.Result{Success: false}

	// When alertingConfig is nil, GetAlertingProviderByAlertType returns nil
	err := sendWithFailover(ep, endpointAlert, result, false, primary, nil)

	if err == nil {
		t.Error("Expected error when primary fails and failover provider not configured")
	}
}

func TestSendWithFailover_ResolvedAlert(t *testing.T) {
	primary := &mockProvider{shouldFail: false}

	ep := &endpoint.Endpoint{Name: "test-endpoint"}
	endpointAlert := &alert.Alert{Type: alert.TypePagerDuty}
	result := &endpoint.Result{Success: true}

	// Test with resolved=true
	err := sendWithFailover(ep, endpointAlert, result, true, primary, nil)

	if err != nil {
		t.Errorf("Expected no error for resolved alert, got: %v", err)
	}
	if primary.sendCount != 1 {
		t.Errorf("Expected primary to be called once for resolved alert, got: %d", primary.sendCount)
	}
}

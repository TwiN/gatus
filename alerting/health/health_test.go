package health

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
)

func TestSanitizeError(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		Name     string
		Input    string
		Contains string
		Excludes string
	}{
		{
			Name:     "redacts-url",
			Input:    "post failed for https://hooks.slack.com/services/XXX/YYY/ZZZ",
			Contains: "[redacted-url]",
			Excludes: "hooks.slack.com",
		},
		{
			Name:     "truncates-long",
			Input:    strings.Repeat("a", 300),
			Excludes: strings.Repeat("a", 300),
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeError(scenario.Input)
			if scenario.Contains != "" && !strings.Contains(got, scenario.Contains) {
				t.Errorf("expected %q to contain %q", got, scenario.Contains)
			}
			if scenario.Excludes != "" && strings.Contains(got, scenario.Excludes) {
				t.Errorf("expected %q not to contain %q", got, scenario.Excludes)
			}
		})
	}
}

func TestGetSnapshot_healthyAndUnhealthy(t *testing.T) {
	t.Parallel()
	ResetAndSetProviders([]ProviderStatus{
		{Type: alert.TypeSlack, Status: ProviderStatusOK},
	})
	SetEndpointWarnings(nil)
	if snap := GetSnapshot(); !snap.Healthy {
		t.Error("expected healthy snapshot")
	}
	RecordFailure(alert.TypeSlack, errors.New("status 404"))
	snap := GetSnapshot()
	if snap.Healthy {
		t.Error("expected unhealthy after delivery failure")
	}
	if len(snap.Delivery) != 1 || snap.Delivery[0].FailureCount != 1 {
		t.Errorf("unexpected delivery: %+v", snap.Delivery)
	}
	RecordSuccess(alert.TypeSlack)
	if snap := GetSnapshot(); !snap.Healthy {
		t.Error("expected healthy after success clears delivery")
	}
}

func TestResetClearsDelivery(t *testing.T) {
	t.Parallel()
	RecordFailure(alert.TypeDiscord, errors.New("err"))
	ResetAndSetProviders(nil)
	snap := GetSnapshot()
	if len(snap.Delivery) != 0 {
		t.Error("expected delivery cleared on reset")
	}
}

func TestProviderIgnoredUnhealthy(t *testing.T) {
	t.Parallel()
	ResetAndSetProviders([]ProviderStatus{
		{Type: alert.TypeSlack, Status: ProviderStatusIgnored, Reason: "webhook-url not set"},
	})
	SetEndpointWarnings(nil)
	if snap := GetSnapshot(); snap.Healthy {
		t.Error("expected unhealthy when provider ignored")
	}
}

func TestRecordFailureTimestamp(t *testing.T) {
	t.Parallel()
	ResetAndSetProviders(nil)
	before := time.Now().UTC()
	RecordFailure(alert.TypeTelegram, errors.New("fail"))
	after := time.Now().UTC()
	snap := GetSnapshot()
	if len(snap.Delivery) != 1 {
		t.Fatal("expected one delivery status")
	}
	if snap.Delivery[0].LastFailureAt.Before(before) || snap.Delivery[0].LastFailureAt.After(after) {
		t.Errorf("unexpected last failure time: %v", snap.Delivery[0].LastFailureAt)
	}
}
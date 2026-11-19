package health

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
)

const (
	ProviderStatusOK           = "ok"
	ProviderStatusIgnored      = "ignored"
	ProviderStatusUnavailable  = "unavailable"
	maxSanitizedErrorRunes     = 256
)

var (
	urlPattern     = regexp.MustCompile(`(?i)https?://[^\s'"<>]+`)
	bearerPattern  = regexp.MustCompile(`(?i)bearer\s+[a-z0-9\-._~+/]+=*`)
	tokenPattern   = regexp.MustCompile(`(?i)(api[_-]?key|token|secret|webhook)[=:\s]+[^\s,;'"]+`)

	mu sync.RWMutex

	providerStatuses []ProviderStatus
	endpointWarnings []string
	deliveryByType   map[alert.Type]*deliveryRecord
)

type deliveryRecord struct {
	lastError     string
	lastFailureAt time.Time
	failureCount  int
	hasFailure    bool
}

// ProviderStatus describes boot-time validation of an alerting provider.
type ProviderStatus struct {
	Type   alert.Type `json:"type"`
	Status string     `json:"status"`
	Reason string     `json:"reason,omitempty"`
}

// DeliveryStatus describes runtime alert delivery failures for a provider type.
type DeliveryStatus struct {
	Type          alert.Type `json:"type"`
	LastError     string     `json:"lastError,omitempty"`
	LastFailureAt time.Time  `json:"lastFailureAt,omitempty"`
	FailureCount  int        `json:"failureCount"`
}

// Snapshot is the alerting health exposed to authenticated UI clients.
type Snapshot struct {
	Healthy   bool             `json:"healthy"`
	Providers []ProviderStatus `json:"providers"`
	Delivery  []DeliveryStatus `json:"delivery"`
	Warnings  []string         `json:"warnings,omitempty"`
}

// ResetAndSetProviders replaces boot-time provider status and clears runtime delivery state.
func ResetAndSetProviders(providers []ProviderStatus) {
	mu.Lock()
	defer mu.Unlock()
	providerStatuses = append([]ProviderStatus(nil), providers...)
	sort.Slice(providerStatuses, func(i, j int) bool {
		return providerStatuses[i].Type < providerStatuses[j].Type
	})
	deliveryByType = make(map[alert.Type]*deliveryRecord)
}

// SetEndpointWarnings sets warnings for endpoints that reference unavailable providers.
func SetEndpointWarnings(warnings []string) {
	mu.Lock()
	defer mu.Unlock()
	endpointWarnings = append([]string(nil), warnings...)
}

// RecordFailure records a failed alert delivery for the given provider type.
func RecordFailure(alertType alert.Type, err error) {
	if alertType == "" {
		return
	}
	msg := ""
	if err != nil {
		msg = SanitizeError(err.Error())
	}
	now := time.Now().UTC()
	mu.Lock()
	defer mu.Unlock()
	if deliveryByType == nil {
		deliveryByType = make(map[alert.Type]*deliveryRecord)
	}
	rec, ok := deliveryByType[alertType]
	if !ok {
		rec = &deliveryRecord{}
		deliveryByType[alertType] = rec
	}
	rec.lastError = msg
	rec.lastFailureAt = now
	rec.failureCount++
	rec.hasFailure = true
}

// RecordSuccess clears runtime delivery failure state for the given provider type.
func RecordSuccess(alertType alert.Type) {
	if alertType == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	delete(deliveryByType, alertType)
}

// GetSnapshot returns the current alerting health.
func GetSnapshot() Snapshot {
	mu.RLock()
	defer mu.RUnlock()
	providers := append([]ProviderStatus(nil), providerStatuses...)
	warnings := append([]string(nil), endpointWarnings...)
	var delivery []DeliveryStatus
	for alertType, rec := range deliveryByType {
		if rec == nil || !rec.hasFailure {
			continue
		}
		delivery = append(delivery, DeliveryStatus{
			Type:          alertType,
			LastError:     rec.lastError,
			LastFailureAt: rec.lastFailureAt,
			FailureCount:  rec.failureCount,
		})
	}
	sort.Slice(delivery, func(i, j int) bool {
		return delivery[i].Type < delivery[j].Type
	})
	healthy := len(warnings) == 0 && len(delivery) == 0
	for _, p := range providers {
		if p.Status != ProviderStatusOK {
			healthy = false
			break
		}
	}
	return Snapshot{
		Healthy:   healthy,
		Providers: providers,
		Delivery:  delivery,
		Warnings:  warnings,
	}
}

// SanitizeError redacts URLs and common secret patterns from error messages.
func SanitizeError(msg string) string {
	if msg == "" {
		return msg
	}
	s := urlPattern.ReplaceAllString(msg, "[redacted-url]")
	s = bearerPattern.ReplaceAllString(s, "bearer [redacted]")
	s = tokenPattern.ReplaceAllString(s, "$1 [redacted]")
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > maxSanitizedErrorRunes {
		s = string(runes[:maxSanitizedErrorRunes]) + "…"
	}
	return s
}

// FormatEndpointWarning builds a single aggregated warning for an alert type.
func FormatEndpointWarning(alertType alert.Type, endpointCount int) string {
	return fmt.Sprintf("%d endpoint(s) reference alert type %s but the provider is not configured", endpointCount, alertType)
}
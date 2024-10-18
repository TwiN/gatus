package memory

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
)

func BenchmarkProcessUptimeAfterResult(b *testing.B) {
	uptime := endpoint.NewUptime()
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	// Start 12000 days ago
	timestamp := now.Add(-12000 * 24 * time.Hour)
	for n := 0; n < b.N; n++ {
		processUptimeAfterResult(uptime, &endpoint.Result{
			Duration:  18 * time.Millisecond,
			Success:   n%15 == 0,
			Timestamp: timestamp,
		})
		// Simulate an endpoint with an interval of 3 minutes
		timestamp = timestamp.Add(3 * time.Minute)
	}
	b.ReportAllocs()
}

package memory

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
)

func TestProcessUptimeAfterResult(t *testing.T) {
	ep := &endpoint.Endpoint{Name: "name", Group: "group"}
	status := endpoint.NewStatus(ep.Group, ep.Name)
	uptime := status.Uptime

	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-7 * 24 * time.Hour), Success: true})

	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-6 * 24 * time.Hour), Success: false})

	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-8 * 24 * time.Hour), Success: true})

	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-24 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-12 * time.Hour), Success: true})

	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-1 * time.Hour), Success: true, Duration: 10 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 10, 1, 1)
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-30 * time.Minute), Success: false, Duration: 500 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 510, 2, 1)
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-15 * time.Minute), Success: false, Duration: 25 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 535, 3, 1)

	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-10 * time.Minute), Success: false})

	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-120 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-119 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-118 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-117 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-10 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-8 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-30 * time.Minute), Success: true})
	processUptimeAfterResult(uptime, &endpoint.Result{Timestamp: now.Add(-25 * time.Minute), Success: true})
}

func TestAddResultUptimeIsCleaningUpAfterItself(t *testing.T) {
	ep := &endpoint.Endpoint{Name: "name", Group: "group"}
	status := endpoint.NewStatus(ep.Group, ep.Name)
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	// Start 12 days ago
	timestamp := now.Add(-12 * 24 * time.Hour)
	for timestamp.Unix() <= now.Unix() {
		AddResult(status, &endpoint.Result{Timestamp: timestamp, Success: true}, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
		if len(status.Uptime.HourlyStatistics) > uptimeCleanUpThreshold {
			t.Errorf("At no point in time should there be more than %d entries in status.SuccessfulExecutionsPerHour, but there are %d", uptimeCleanUpThreshold, len(status.Uptime.HourlyStatistics))
		}
		// Simulate endpoint with an interval of 3 minutes
		timestamp = timestamp.Add(3 * time.Minute)
	}
}

func checkHourlyStatistics(t *testing.T, hourlyUptimeStatistics *endpoint.HourlyUptimeStatistics, expectedTotalExecutionsResponseTime uint64, expectedTotalExecutions uint64, expectedSuccessfulExecutions uint64) {
	if hourlyUptimeStatistics.TotalExecutionsResponseTime != expectedTotalExecutionsResponseTime {
		t.Error("TotalExecutionsResponseTime should've been", expectedTotalExecutionsResponseTime, "got", hourlyUptimeStatistics.TotalExecutionsResponseTime)
	}
	if hourlyUptimeStatistics.TotalExecutions != expectedTotalExecutions {
		t.Error("TotalExecutions should've been", expectedTotalExecutions, "got", hourlyUptimeStatistics.TotalExecutions)
	}
	if hourlyUptimeStatistics.SuccessfulExecutions != expectedSuccessfulExecutions {
		t.Error("SuccessfulExecutions should've been", expectedSuccessfulExecutions, "got", hourlyUptimeStatistics.SuccessfulExecutions)
	}
}

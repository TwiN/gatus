package memory

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
)

func TestProcessUptimeAfterResult(t *testing.T) {
	service := &core.Service{Name: "name", Group: "group"}
	serviceStatus := core.NewServiceStatus(service.Key(), service.Group, service.Name)
	uptime := serviceStatus.Uptime

	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-7 * 24 * time.Hour), Success: true})

	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-6 * 24 * time.Hour), Success: false})

	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-8 * 24 * time.Hour), Success: true})

	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-24 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-12 * time.Hour), Success: true})

	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-1 * time.Hour), Success: true, Duration: 10 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 10, 1, 1)
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-30 * time.Minute), Success: false, Duration: 500 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 510, 2, 1)
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-15 * time.Minute), Success: false, Duration: 25 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 535, 3, 1)

	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-10 * time.Minute), Success: false})

	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-120 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-119 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-118 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-117 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-10 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-8 * time.Hour), Success: true})
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-30 * time.Minute), Success: true})
	processUptimeAfterResult(uptime, &core.Result{Timestamp: now.Add(-25 * time.Minute), Success: true})
}

func TestAddResultUptimeIsCleaningUpAfterItself(t *testing.T) {
	service := &core.Service{Name: "name", Group: "group"}
	serviceStatus := core.NewServiceStatus(service.Key(), service.Group, service.Name)
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	// Start 12 days ago
	timestamp := now.Add(-12 * 24 * time.Hour)
	for timestamp.Unix() <= now.Unix() {
		AddResult(serviceStatus, &core.Result{Timestamp: timestamp, Success: true})
		if len(serviceStatus.Uptime.HourlyStatistics) > numberOfHoursInTenDays {
			t.Errorf("At no point in time should there be more than %d entries in serviceStatus.SuccessfulExecutionsPerHour, but there are %d", numberOfHoursInTenDays, len(serviceStatus.Uptime.HourlyStatistics))
		}
		// Simulate service with an interval of 3 minutes
		timestamp = timestamp.Add(3 * time.Minute)
	}
}

func checkHourlyStatistics(t *testing.T, hourlyUptimeStatistics *core.HourlyUptimeStatistics, expectedTotalExecutionsResponseTime uint64, expectedTotalExecutions uint64, expectedSuccessfulExecutions uint64) {
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

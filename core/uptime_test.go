package core

import (
	"testing"
	"time"
)

func TestUptime_ProcessResult(t *testing.T) {
	service := &Service{Name: "name", Group: "group"}
	serviceStatus := NewServiceStatus(service.Key(), service.Group, service.Name)
	uptime := serviceStatus.Uptime

	checkUptimes(t, serviceStatus, 0.00, 0.00, 0.00)

	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	uptime.ProcessResult(&Result{Timestamp: now.Add(-7 * 24 * time.Hour), Success: true})
	checkUptimes(t, serviceStatus, 1.00, 0.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: now.Add(-6 * 24 * time.Hour), Success: false})
	checkUptimes(t, serviceStatus, 0.50, 0.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: now.Add(-8 * 24 * time.Hour), Success: true})
	checkUptimes(t, serviceStatus, 0.50, 0.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: now.Add(-24 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: now.Add(-12 * time.Hour), Success: true})
	checkUptimes(t, serviceStatus, 0.75, 1.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: now.Add(-1 * time.Hour), Success: true, Duration: 10 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 10, 1, 1)
	uptime.ProcessResult(&Result{Timestamp: now.Add(-30 * time.Minute), Success: false, Duration: 500 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 510, 2, 1)
	uptime.ProcessResult(&Result{Timestamp: now.Add(-15 * time.Minute), Success: false, Duration: 25 * time.Millisecond})
	checkHourlyStatistics(t, uptime.HourlyStatistics[now.Unix()-now.Unix()%3600-3600], 535, 3, 1)

	uptime.ProcessResult(&Result{Timestamp: now.Add(-10 * time.Minute), Success: false})
	checkUptimes(t, serviceStatus, 0.50, 0.50, 0.25)

	uptime.ProcessResult(&Result{Timestamp: now.Add(-120 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: now.Add(-119 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: now.Add(-118 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: now.Add(-117 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: now.Add(-10 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: now.Add(-8 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: now.Add(-30 * time.Minute), Success: true})
	uptime.ProcessResult(&Result{Timestamp: now.Add(-25 * time.Minute), Success: true})
	checkUptimes(t, serviceStatus, 0.75, 0.70, 0.50)
}

func TestServiceStatus_AddResultUptimeIsCleaningUpAfterItself(t *testing.T) {
	service := &Service{Name: "name", Group: "group"}
	serviceStatus := NewServiceStatus(service.Key(), service.Group, service.Name)
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	// Start 12 days ago
	timestamp := now.Add(-12 * 24 * time.Hour)
	for timestamp.Unix() <= now.Unix() {
		serviceStatus.AddResult(&Result{Timestamp: timestamp, Success: true})
		if len(serviceStatus.Uptime.HourlyStatistics) > numberOfHoursInTenDays {
			t.Errorf("At no point in time should there be more than %d entries in serviceStatus.SuccessfulExecutionsPerHour, but there are %d", numberOfHoursInTenDays, len(serviceStatus.Uptime.HourlyStatistics))
		}
		if now.Sub(timestamp) > time.Hour && serviceStatus.Uptime.LastHour != 0 {
			t.Error("most recent timestamp > 1h ago, expected serviceStatus.Uptime.LastHour to be 0, got", serviceStatus.Uptime.LastHour)
		}
		if now.Sub(timestamp) < time.Hour && serviceStatus.Uptime.LastHour == 0 {
			t.Error("most recent timestamp < 1h ago, expected serviceStatus.Uptime.LastHour to NOT be 0, got", serviceStatus.Uptime.LastHour)
		}
		// Simulate service with an interval of 3 minutes
		timestamp = timestamp.Add(3 * time.Minute)
	}
}

func checkUptimes(t *testing.T, status *ServiceStatus, expectedUptimeDuringLastSevenDays, expectedUptimeDuringLastTwentyFourHours, expectedUptimeDuringLastHour float64) {
	if status.Uptime.LastSevenDays != expectedUptimeDuringLastSevenDays {
		t.Errorf("expected status.Uptime.LastSevenDays to be %f, got %f", expectedUptimeDuringLastHour, status.Uptime.LastSevenDays)
	}
	if status.Uptime.LastTwentyFourHours != expectedUptimeDuringLastTwentyFourHours {
		t.Errorf("expected status.Uptime.LastTwentyFourHours to be %f, got %f", expectedUptimeDuringLastTwentyFourHours, status.Uptime.LastTwentyFourHours)
	}
	if status.Uptime.LastHour != expectedUptimeDuringLastHour {
		t.Errorf("expected status.Uptime.LastHour to be %f, got %f", expectedUptimeDuringLastHour, status.Uptime.LastHour)
	}
}

func checkHourlyStatistics(t *testing.T, hourlyUptimeStatistics *HourlyUptimeStatistics, expectedTotalExecutionsResponseTime uint64, expectedTotalExecutions uint64, expectedSuccessfulExecutions uint64) {
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

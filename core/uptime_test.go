package core

import (
	"testing"
	"time"
)

func TestUptime_ProcessResult(t *testing.T) {
	service := &Service{Name: "name", Group: "group"}
	serviceStatus := NewServiceStatus(service)
	uptime := serviceStatus.Uptime

	checkUptimes(t, serviceStatus, 0.00, 0.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-7 * 24 * time.Hour), Success: true})
	checkUptimes(t, serviceStatus, 1.00, 0.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-6 * 24 * time.Hour), Success: false})
	checkUptimes(t, serviceStatus, 0.50, 0.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-8 * 24 * time.Hour), Success: true})
	checkUptimes(t, serviceStatus, 0.50, 0.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-24 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-12 * time.Hour), Success: true})
	checkUptimes(t, serviceStatus, 0.75, 1.00, 0.00)

	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-1 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-30 * time.Minute), Success: false})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-15 * time.Minute), Success: false})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-10 * time.Minute), Success: false})
	checkUptimes(t, serviceStatus, 0.50, 0.50, 0.25)

	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-120 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-119 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-118 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-117 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-10 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-8 * time.Hour), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-30 * time.Minute), Success: true})
	uptime.ProcessResult(&Result{Timestamp: time.Now().Add(-25 * time.Minute), Success: true})
	checkUptimes(t, serviceStatus, 0.75, 0.70, 0.50)
}

func TestServiceStatus_AddResultUptimeIsCleaningUpAfterItself(t *testing.T) {
	service := &Service{Name: "name", Group: "group"}
	serviceStatus := NewServiceStatus(service)
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	// Start 12 days ago
	timestamp := now.Add(-12 * 24 * time.Hour)
	for timestamp.Unix() <= now.Unix() {
		serviceStatus.AddResult(&Result{Timestamp: timestamp, Success: true})
		if len(serviceStatus.Uptime.SuccessCountPerHour) > numberOfHoursInTenDays {
			t.Errorf("At no point in time should there be more than %d entries in serviceStatus.SuccessCountPerHour", numberOfHoursInTenDays)
		}
		//fmt.Printf("timestamp=%s; uptimeDuringLastHour=%f; timeAgo=%s\n", timestamp.Format(time.RFC3339), serviceStatus.UptimeDuringLastHour, time.Since(timestamp))
		if now.Sub(timestamp) > time.Hour && serviceStatus.Uptime.LastHour != 0 {
			t.Error("most recent timestamp > 1h ago, expected serviceStatus.Uptime.LastHour to be 0, got", serviceStatus.Uptime.LastHour)
		}
		if now.Sub(timestamp) < time.Hour && serviceStatus.Uptime.LastHour == 0 {
			t.Error("most recent timestamp < 1h ago, expected serviceStatus.Uptime.LastHour to NOT be 0, got", serviceStatus.Uptime.LastHour)
		}
		// Simulate service with an interval of 1 minute
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

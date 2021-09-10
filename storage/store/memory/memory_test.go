package memory

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/common/paging"
)

var (
	firstCondition  = core.Condition("[STATUS] == 200")
	secondCondition = core.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = core.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	now = time.Now()

	testService = core.Service{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []*core.Condition{&firstCondition, &secondCondition, &thirdCondition},
		Alerts:                  nil,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuccessfulResult = core.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                nil,
		Connected:             true,
		Success:               true,
		Timestamp:             now,
		Duration:              150 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*core.ConditionResult{
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
			{
				Condition: "[RESPONSE_TIME] < 500",
				Success:   true,
			},
			{
				Condition: "[CERTIFICATE_EXPIRATION] < 72h",
				Success:   true,
			},
		},
	}
	testUnsuccessfulResult = core.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                []string{"error-1", "error-2"},
		Connected:             true,
		Success:               false,
		Timestamp:             now,
		Duration:              750 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*core.ConditionResult{
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
			{
				Condition: "[RESPONSE_TIME] < 500",
				Success:   false,
			},
			{
				Condition: "[CERTIFICATE_EXPIRATION] < 72h",
				Success:   false,
			},
		},
	}
)

// Note that are much more extensive tests in /storage/store/store_test.go.
// This test is simply an extra sanity check
func TestStore_SanityCheck(t *testing.T) {
	store, _ := NewStore("")
	defer store.Close()
	store.Insert(&testService, &testSuccessfulResult)
	serviceStatuses, _ := store.GetAllServiceStatuses(paging.NewServiceStatusParams())
	if numberOfServiceStatuses := len(serviceStatuses); numberOfServiceStatuses != 1 {
		t.Fatalf("expected 1 ServiceStatus, got %d", numberOfServiceStatuses)
	}
	store.Insert(&testService, &testUnsuccessfulResult)
	// Both results inserted are for the same service, therefore, the count shouldn't have increased
	serviceStatuses, _ = store.GetAllServiceStatuses(paging.NewServiceStatusParams())
	if numberOfServiceStatuses := len(serviceStatuses); numberOfServiceStatuses != 1 {
		t.Fatalf("expected 1 ServiceStatus, got %d", numberOfServiceStatuses)
	}
	if hourlyAverageResponseTime, err := store.GetHourlyAverageResponseTimeByKey(testService.Key(), time.Now().Add(-24*time.Hour), time.Now()); err != nil {
		t.Errorf("expected no error, got %v", err)
	} else if len(hourlyAverageResponseTime) != 1 {
		t.Errorf("expected 1 hour to have had a result in the past 24 hours, got %d", len(hourlyAverageResponseTime))
	}
	if uptime, _ := store.GetUptimeByKey(testService.Key(), time.Now().Add(-24*time.Hour), time.Now()); uptime != 0.5 {
		t.Errorf("expected uptime of last 24h to be 0.5, got %f", uptime)
	}
	if averageResponseTime, _ := store.GetAverageResponseTimeByKey(testService.Key(), time.Now().Add(-24*time.Hour), time.Now()); averageResponseTime != 450 {
		t.Errorf("expected average response time of last 24h to be 450, got %d", averageResponseTime)
	}
	ss, _ := store.GetServiceStatus(testService.Group, testService.Name, paging.NewServiceStatusParams().WithResults(1, 20).WithEvents(1, 20))
	if ss == nil {
		t.Fatalf("Store should've had key '%s', but didn't", testService.Key())
	}
	if len(ss.Events) != 3 {
		t.Errorf("Service '%s' should've had 3 events, got %d", ss.Name, len(ss.Events))
	}
	if len(ss.Results) != 2 {
		t.Errorf("Service '%s' should've had 2 results, got %d", ss.Name, len(ss.Results))
	}
	if deleted := store.DeleteAllServiceStatusesNotInKeys([]string{}); deleted != 1 {
		t.Errorf("%d entries should've been deleted, got %d", 1, deleted)
	}
}

func TestStore_Save(t *testing.T) {
	files := []string{
		"",
		t.TempDir() + "/test.db",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			store, err := NewStore(file)
			if err != nil {
				t.Fatal("expected no error, got", err.Error())
			}
			err = store.Save()
			if err != nil {
				t.Fatal("expected no error, got", err.Error())
			}
			store.Clear()
			store.Close()
		})
	}
}

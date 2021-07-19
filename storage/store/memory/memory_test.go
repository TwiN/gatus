package memory

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/paging"
)

var (
	firstCondition  = core.Condition("[STATUS] == 200")
	secondCondition = core.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = core.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	timestamp = time.Now()

	testService = core.Service{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []*core.Condition{&firstCondition, &secondCondition, &thirdCondition},
		Alerts:                  nil,
		Insecure:                false,
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
		Timestamp:             timestamp,
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
		Timestamp:             timestamp,
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

// Note that there is a much more extensive test in /storage/store/store_test.go.
// This test is simply an extra sanity check
func TestStore_Insert(t *testing.T) {
	store, _ := NewStore("")
	store.Insert(&testService, &testSuccessfulResult)
	if numberOfServiceStatuses := len(store.GetAllServiceStatuses(paging.NewServiceStatusParams())); numberOfServiceStatuses != 1 {
		t.Fatalf("expected 1 ServiceStatus, got %d", numberOfServiceStatuses)
	}
	store.Insert(&testService, &testUnsuccessfulResult)
	// Both results inserted are for the same service, therefore, the count shouldn't have increased
	if numberOfServiceStatuses := len(store.GetAllServiceStatuses(paging.NewServiceStatusParams())); numberOfServiceStatuses != 1 {
		t.Fatalf("expected 1 ServiceStatus, got %d", numberOfServiceStatuses)
	}
	ss := store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithResults(1, 20).WithEvents(1, 20))
	if ss == nil {
		t.Fatalf("Store should've had key '%s', but didn't", testService.Key())
	}
	if len(ss.Events) != 3 {
		t.Fatalf("Service '%s' should've had 3 events, got %d", ss.Name, len(ss.Events))
	}
	if len(ss.Results) != 2 {
		t.Fatalf("Service '%s' should've had 2 results, got %d", ss.Name, len(ss.Results))
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
		})
	}
}

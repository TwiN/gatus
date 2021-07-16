package store

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/database"
	"github.com/TwinProduction/gatus/storage/store/memory"
	"github.com/TwinProduction/gatus/storage/store/paging"
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
		Insecure:                false,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuccessfulResult = core.Result{
		Timestamp:             now,
		Success:               true,
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                nil,
		Connected:             true,
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
		Timestamp:             now,
		Success:               false,
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                []string{"error-1", "error-2"},
		Connected:             true,
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

func TestStore_GetServiceStatusByKey(t *testing.T) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", t.TempDir()+"/TestStore_GetServiceStatusByKey.db")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
		{
			Name:  "database",
			Store: databaseStore,
		},
	}
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-time.Minute)
	secondResult := testUnsuccessfulResult
	secondResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &firstResult)
			scenario.Store.Insert(&testService, &secondResult)

			serviceStatus := scenario.Store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
			if serviceStatus == nil {
				t.Fatalf("serviceStatus shouldn't have been nil")
			}
			if serviceStatus.Name != testService.Name {
				t.Fatalf("serviceStatus.Name should've been %s, got %s", testService.Name, serviceStatus.Name)
			}
			if serviceStatus.Group != testService.Group {
				t.Fatalf("serviceStatus.Group should've been %s, got %s", testService.Group, serviceStatus.Group)
			}
			if len(serviceStatus.Results) != 2 {
				t.Fatalf("serviceStatus.Results should've had 2 entries")
			}
			if serviceStatus.Results[0].Timestamp.After(serviceStatus.Results[1].Timestamp) {
				t.Error("The result at index 0 should've been older than the result at index 1")
			}
			if serviceStatus.Uptime == nil {
				t.Fatalf("serviceStatus.Uptime shouldn't have been nil")
			}
			if serviceStatus.Uptime.LastHour != 0.5 {
				t.Errorf("serviceStatus.Uptime.LastHour should've been 0.5, got %f", serviceStatus.Uptime.LastHour)
			}
			if serviceStatus.Uptime.LastTwentyFourHours != 0.5 {
				t.Errorf("serviceStatus.Uptime.LastTwentyFourHours should've been 0.5, got %f", serviceStatus.Uptime.LastTwentyFourHours)
			}
			if serviceStatus.Uptime.LastSevenDays != 0.5 {
				t.Errorf("serviceStatus.Uptime.LastSevenDays should've been 0.5, got %f", serviceStatus.Uptime.LastSevenDays)
			}
			scenario.Store.Clear()
		})
	}
}

func TestStore_GetServiceStatusForMissingStatusReturnsNil(t *testing.T) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", t.TempDir()+"/TestStore_GetServiceStatusForMissingStatusReturnsNil.db")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
		{
			Name:  "database",
			Store: databaseStore,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &testSuccessfulResult)
			serviceStatus := scenario.Store.GetServiceStatus("nonexistantgroup", "nonexistantname", paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
			if serviceStatus != nil {
				t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, testService.Name)
			}
			serviceStatus = scenario.Store.GetServiceStatus(testService.Group, "nonexistantname", paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
			if serviceStatus != nil {
				t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, "nonexistantname")
			}
			serviceStatus = scenario.Store.GetServiceStatus("nonexistantgroup", testService.Name, paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
			if serviceStatus != nil {
				t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", "nonexistantgroup", testService.Name)
			}
		})
	}
}

func TestStore_GetAllServiceStatuses(t *testing.T) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", t.TempDir()+"/TestStore_GetAllServiceStatuses.db")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
		{
			Name:  "database",
			Store: databaseStore,
		},
	}
	firstResult := testSuccessfulResult
	secondResult := testUnsuccessfulResult
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &firstResult)
			scenario.Store.Insert(&testService, &secondResult)
			// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
			serviceStatuses := scenario.Store.GetAllServiceStatuses(paging.NewServiceStatusParams().WithResults(1, 20))
			if len(serviceStatuses) != 1 {
				t.Fatal("expected 1 service status")
			}
			actual, exists := serviceStatuses[testService.Key()]
			if !exists {
				t.Fatal("expected service status to exist")
			}
			if len(actual.Results) != 2 {
				t.Error("expected 2 results, got", len(actual.Results))
			}
			if len(actual.Events) != 0 {
				t.Error("expected 0 events, got", len(actual.Events))
			}
			scenario.Store.Clear()
		})
	}
}

func TestStore_GetAllServiceStatusesWithResultsAndEvents(t *testing.T) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", t.TempDir()+"/TestStore_GetAllServiceStatusesWithResultsAndEvents.db")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
		{
			Name:  "database",
			Store: databaseStore,
		},
	}
	firstResult := testSuccessfulResult
	secondResult := testUnsuccessfulResult
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &firstResult)
			scenario.Store.Insert(&testService, &secondResult)
			// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
			serviceStatuses := scenario.Store.GetAllServiceStatuses(paging.NewServiceStatusParams().WithResults(1, 20).WithEvents(1, 50))
			if len(serviceStatuses) != 1 {
				t.Fatal("expected 1 service status")
			}
			actual, exists := serviceStatuses[testService.Key()]
			if !exists {
				t.Fatal("expected service status to exist")
			}
			if len(actual.Results) != 2 {
				t.Error("expected 2 results, got", len(actual.Results))
			}
			if len(actual.Events) != 3 {
				t.Error("expected 3 events, got", len(actual.Events))
			}
			scenario.Store.Clear()
		})
	}
}

func TestStore_GetServiceStatusPage1IsHasMoreRecentResultsThanPage2(t *testing.T) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", t.TempDir()+"/TestStore_GetServiceStatusPage1IsHasMoreRecentResultsThanPage2.db")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
		{
			Name:  "database",
			Store: databaseStore,
		},
	}
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-time.Minute)
	secondResult := testUnsuccessfulResult
	secondResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &firstResult)
			scenario.Store.Insert(&testService, &secondResult)
			serviceStatusPage1 := scenario.Store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithResults(1, 1))
			if serviceStatusPage1 == nil {
				t.Fatalf("serviceStatusPage1 shouldn't have been nil")
			}
			if len(serviceStatusPage1.Results) != 1 {
				t.Fatalf("serviceStatusPage1 should've had 1 result")
			}
			serviceStatusPage2 := scenario.Store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithResults(2, 1))
			if serviceStatusPage2 == nil {
				t.Fatalf("serviceStatusPage2 shouldn't have been nil")
			}
			if len(serviceStatusPage2.Results) != 1 {
				t.Fatalf("serviceStatusPage2 should've had 1 result")
			}
			// Compare the timestamp of both pages
			if !serviceStatusPage1.Results[0].Timestamp.After(serviceStatusPage2.Results[0].Timestamp) {
				t.Errorf("The result from the first page should've been more recent than the results from the second page")
			}
			scenario.Store.Clear()
		})
	}
}

func TestStore_Insert(t *testing.T) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", t.TempDir()+"/TestStore_Insert.db")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
		{
			Name:  "database",
			Store: databaseStore,
		},
	}
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-time.Minute)
	secondResult := testUnsuccessfulResult
	secondResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &testSuccessfulResult)
			scenario.Store.Insert(&testService, &testUnsuccessfulResult)

			serviceStatus := scenario.Store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
			if serviceStatus == nil {
				t.Fatalf("Store should've had key '%s', but didn't", testService.Key())
			}
			if len(serviceStatus.Results) != 2 {
				t.Fatalf("Service '%s' should've had 2 results, but actually returned %d", serviceStatus.Name, len(serviceStatus.Results))
			}
			for i, r := range serviceStatus.Results {
				expectedResult := scenario.Store.GetServiceStatus(testService.Group, testService.Name, paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime()).Results[i]
				if r.HTTPStatus != expectedResult.HTTPStatus {
					t.Errorf("Result at index %d should've had a HTTPStatus of %d, but was actually %d", i, expectedResult.HTTPStatus, r.HTTPStatus)
				}
				if r.DNSRCode != expectedResult.DNSRCode {
					t.Errorf("Result at index %d should've had a DNSRCode of %s, but was actually %s", i, expectedResult.DNSRCode, r.DNSRCode)
				}
				if r.Hostname != expectedResult.Hostname {
					t.Errorf("Result at index %d should've had a Hostname of %s, but was actually %s", i, expectedResult.Hostname, r.Hostname)
				}
				if r.IP != expectedResult.IP {
					t.Errorf("Result at index %d should've had a IP of %s, but was actually %s", i, expectedResult.IP, r.IP)
				}
				if r.Connected != expectedResult.Connected {
					t.Errorf("Result at index %d should've had a Connected value of %t, but was actually %t", i, expectedResult.Connected, r.Connected)
				}
				if r.Duration != expectedResult.Duration {
					t.Errorf("Result at index %d should've had a Duration of %s, but was actually %s", i, expectedResult.Duration.String(), r.Duration.String())
				}
				if len(r.Errors) != len(expectedResult.Errors) {
					t.Errorf("Result at index %d should've had %d errors, but actually had %d errors", i, len(expectedResult.Errors), len(r.Errors))
				}
				if len(r.ConditionResults) != len(expectedResult.ConditionResults) {
					t.Errorf("Result at index %d should've had %d ConditionResults, but actually had %d ConditionResults", i, len(expectedResult.ConditionResults), len(r.ConditionResults))
				}
				if r.Success != expectedResult.Success {
					t.Errorf("Result at index %d should've had a Success of %t, but was actually %t", i, expectedResult.Success, r.Success)
				}
				if r.Timestamp != expectedResult.Timestamp {
					t.Errorf("Result at index %d should've had a Timestamp of %s, but was actually %s", i, expectedResult.Timestamp.String(), r.Timestamp.String())
				}
				if r.CertificateExpiration != expectedResult.CertificateExpiration {
					t.Errorf("Result at index %d should've had a CertificateExpiration of %s, but was actually %s", i, expectedResult.CertificateExpiration.String(), r.CertificateExpiration.String())
				}
			}
		})
	}
}

func TestStore_DeleteAllServiceStatusesNotInKeys(t *testing.T) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", t.TempDir()+"/TestStore_DeleteAllServiceStatusesNotInKeys.db")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
		{
			Name:  "database",
			Store: databaseStore,
		},
	}
	firstService := core.Service{Name: "service-1", Group: "group"}
	secondService := core.Service{Name: "service-2", Group: "group"}
	result := &testSuccessfulResult
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&firstService, result)
			scenario.Store.Insert(&secondService, result)
			if scenario.Store.GetServiceStatusByKey(firstService.Key(), paging.NewServiceStatusParams()) == nil {
				t.Fatal("firstService should exist")
			}
			if scenario.Store.GetServiceStatusByKey(secondService.Key(), paging.NewServiceStatusParams()) == nil {
				t.Fatal("secondService should exist")
			}
			scenario.Store.DeleteAllServiceStatusesNotInKeys([]string{firstService.Key()})
			if scenario.Store.GetServiceStatusByKey(firstService.Key(), paging.NewServiceStatusParams()) == nil {
				t.Error("secondService should've been deleted")
			}
			if scenario.Store.GetServiceStatusByKey(secondService.Key(), paging.NewServiceStatusParams()) != nil {
				t.Error("firstService should still exist")
			}
			scenario.Store.Clear()
		})
	}
}

package store

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/common"
	"github.com/TwinProduction/gatus/storage/store/common/paging"
	"github.com/TwinProduction/gatus/storage/store/memory"
	"github.com/TwinProduction/gatus/storage/store/sqlite"
)

var (
	firstCondition  = core.Condition("[STATUS] == 200")
	secondCondition = core.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = core.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	now = time.Now().Truncate(time.Hour)

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

type Scenario struct {
	Name  string
	Store Store
}

func initStoresAndBaseScenarios(t *testing.T, testName string) []*Scenario {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sqlite.NewStore("sqlite", t.TempDir()+"/"+testName+".db")
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	return []*Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
		{
			Name:  "sqlite",
			Store: sqliteStore,
		},
	}
}

func cleanUp(scenarios []*Scenario) {
	for _, scenario := range scenarios {
		scenario.Store.Close()
	}
}

func TestStore_GetServiceStatusByKey(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetServiceStatusByKey")
	defer cleanUp(scenarios)
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-time.Minute)
	secondResult := testUnsuccessfulResult
	secondResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &firstResult)
			scenario.Store.Insert(&testService, &secondResult)

			serviceStatus := scenario.Store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithEvents(1, common.MaximumNumberOfEvents).WithResults(1, common.MaximumNumberOfResults))
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
			scenario.Store.Clear()
		})
	}
}

func TestStore_GetServiceStatusForMissingStatusReturnsNil(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetServiceStatusForMissingStatusReturnsNil")
	defer cleanUp(scenarios)
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &testSuccessfulResult)
			serviceStatus := scenario.Store.GetServiceStatus("nonexistantgroup", "nonexistantname", paging.NewServiceStatusParams().WithEvents(1, common.MaximumNumberOfEvents).WithResults(1, common.MaximumNumberOfResults))
			if serviceStatus != nil {
				t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, testService.Name)
			}
			serviceStatus = scenario.Store.GetServiceStatus(testService.Group, "nonexistantname", paging.NewServiceStatusParams().WithEvents(1, common.MaximumNumberOfEvents).WithResults(1, common.MaximumNumberOfResults))
			if serviceStatus != nil {
				t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, "nonexistantname")
			}
			serviceStatus = scenario.Store.GetServiceStatus("nonexistantgroup", testService.Name, paging.NewServiceStatusParams().WithEvents(1, common.MaximumNumberOfEvents).WithResults(1, common.MaximumNumberOfResults))
			if serviceStatus != nil {
				t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", "nonexistantgroup", testService.Name)
			}
		})
	}
}

func TestStore_GetAllServiceStatuses(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetAllServiceStatuses")
	defer cleanUp(scenarios)
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
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetAllServiceStatusesWithResultsAndEvents")
	defer cleanUp(scenarios)
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
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetServiceStatusPage1IsHasMoreRecentResultsThanPage2")
	defer cleanUp(scenarios)
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

func TestStore_GetUptimeByKey(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetUptimeByKey")
	defer cleanUp(scenarios)
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-time.Minute)
	secondResult := testUnsuccessfulResult
	secondResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if _, err := scenario.Store.GetUptimeByKey(testService.Key(), time.Now().Add(-time.Hour), time.Now()); err != common.ErrServiceNotFound {
				t.Errorf("should've returned not found because there's nothing yet, got %v", err)
			}
			scenario.Store.Insert(&testService, &firstResult)
			scenario.Store.Insert(&testService, &secondResult)
			if uptime, _ := scenario.Store.GetUptimeByKey(testService.Key(), now.Add(-time.Hour), time.Now()); uptime != 0.5 {
				t.Errorf("the uptime over the past 1h should've been 0.5, got %f", uptime)
			}
			if uptime, _ := scenario.Store.GetUptimeByKey(testService.Key(), now.Add(-time.Hour*24), time.Now()); uptime != 0.5 {
				t.Errorf("the uptime over the past 24h should've been 0.5, got %f", uptime)
			}
			if uptime, _ := scenario.Store.GetUptimeByKey(testService.Key(), now.Add(-time.Hour*24*7), time.Now()); uptime != 0.5 {
				t.Errorf("the uptime over the past 7d should've been 0.5, got %f", uptime)
			}
			if _, err := scenario.Store.GetUptimeByKey(testService.Key(), now, time.Now().Add(-time.Hour)); err == nil {
				t.Error("should've returned an error because the parameter 'from' cannot be older than 'to'")
			}
		})
	}
}

func TestStore_GetAverageResponseTimeByKey(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetAverageResponseTimeByKey")
	defer cleanUp(scenarios)
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-(2 * time.Hour))
	firstResult.Duration = 300 * time.Millisecond
	secondResult := testSuccessfulResult
	secondResult.Duration = 150 * time.Millisecond
	secondResult.Timestamp = now.Add(-(1*time.Hour + 30*time.Minute))
	thirdResult := testUnsuccessfulResult
	thirdResult.Duration = 200 * time.Millisecond
	thirdResult.Timestamp = now.Add(-(1 * time.Hour))
	fourthResult := testSuccessfulResult
	fourthResult.Duration = 500 * time.Millisecond
	fourthResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &firstResult)
			scenario.Store.Insert(&testService, &secondResult)
			scenario.Store.Insert(&testService, &thirdResult)
			scenario.Store.Insert(&testService, &fourthResult)
			if averageResponseTime, err := scenario.Store.GetAverageResponseTimeByKey(testService.Key(), now.Add(-48*time.Hour), now.Add(-24*time.Hour)); err == nil {
				if averageResponseTime != 0 {
					t.Errorf("expected average response time to be 0ms, got %v", averageResponseTime)
				}
			} else {
				t.Error("shouldn't have returned an error, got", err)
			}
			if averageResponseTime, err := scenario.Store.GetAverageResponseTimeByKey(testService.Key(), now.Add(-24*time.Hour), now); err == nil {
				if averageResponseTime != 287 {
					t.Errorf("expected average response time to be 287ms, got %v", averageResponseTime)
				}
			} else {
				t.Error("shouldn't have returned an error, got", err)
			}
			if averageResponseTime, err := scenario.Store.GetAverageResponseTimeByKey(testService.Key(), now.Add(-time.Hour), now); err == nil {
				if averageResponseTime != 350 {
					t.Errorf("expected average response time to be 350ms, got %v", averageResponseTime)
				}
			} else {
				t.Error("shouldn't have returned an error, got", err)
			}
			if averageResponseTime, err := scenario.Store.GetAverageResponseTimeByKey(testService.Key(), now.Add(-2*time.Hour), now.Add(-time.Hour)); err == nil {
				if averageResponseTime != 216 {
					t.Errorf("expected average response time to be 216ms, got %v", averageResponseTime)
				}
			} else {
				t.Error("shouldn't have returned an error, got", err)
			}
			if _, err := scenario.Store.GetAverageResponseTimeByKey(testService.Key(), now, now.Add(-2*time.Hour)); err == nil {
				t.Error("expected an error because from > to, got nil")
			}
			scenario.Store.Clear()
		})
	}
}

func TestStore_GetHourlyAverageResponseTimeByKey(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetHourlyAverageResponseTimeByKey")
	defer cleanUp(scenarios)
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-(2 * time.Hour))
	firstResult.Duration = 300 * time.Millisecond
	secondResult := testSuccessfulResult
	secondResult.Duration = 150 * time.Millisecond
	secondResult.Timestamp = now.Add(-(1*time.Hour + 30*time.Minute))
	thirdResult := testUnsuccessfulResult
	thirdResult.Duration = 200 * time.Millisecond
	thirdResult.Timestamp = now.Add(-(1 * time.Hour))
	fourthResult := testSuccessfulResult
	fourthResult.Duration = 500 * time.Millisecond
	fourthResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &firstResult)
			scenario.Store.Insert(&testService, &secondResult)
			scenario.Store.Insert(&testService, &thirdResult)
			scenario.Store.Insert(&testService, &fourthResult)
			hourlyAverageResponseTime, err := scenario.Store.GetHourlyAverageResponseTimeByKey(testService.Key(), now.Add(-24*time.Hour), now)
			if err != nil {
				t.Error("shouldn't have returned an error, got", err)
			}
			if key := now.Truncate(time.Hour).Unix(); hourlyAverageResponseTime[key] != 500 {
				t.Errorf("expected average response time to be 500ms at %d, got %v", key, hourlyAverageResponseTime[key])
			}
			if key := now.Truncate(time.Hour).Add(-time.Hour).Unix(); hourlyAverageResponseTime[key] != 200 {
				t.Errorf("expected average response time to be 200ms at %d, got %v", key, hourlyAverageResponseTime[key])
			}
			if key := now.Truncate(time.Hour).Add(-2 * time.Hour).Unix(); hourlyAverageResponseTime[key] != 225 {
				t.Errorf("expected average response time to be 225ms at %d, got %v", key, hourlyAverageResponseTime[key])
			}
			scenario.Store.Clear()
		})
	}
}

func TestStore_Insert(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_Insert")
	defer cleanUp(scenarios)
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-time.Minute)
	secondResult := testUnsuccessfulResult
	secondResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.Insert(&testService, &testSuccessfulResult)
			scenario.Store.Insert(&testService, &testUnsuccessfulResult)

			ss := scenario.Store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithEvents(1, common.MaximumNumberOfEvents).WithResults(1, common.MaximumNumberOfResults))
			if ss == nil {
				t.Fatalf("Store should've had key '%s', but didn't", testService.Key())
			}
			if len(ss.Events) != 3 {
				t.Fatalf("Service '%s' should've had 3 events, got %d", ss.Name, len(ss.Events))
			}
			if len(ss.Results) != 2 {
				t.Fatalf("Service '%s' should've had 2 results, got %d", ss.Name, len(ss.Results))
			}
			for i, expectedResult := range []core.Result{testSuccessfulResult, testUnsuccessfulResult} {
				if expectedResult.HTTPStatus != ss.Results[i].HTTPStatus {
					t.Errorf("Result at index %d should've had a HTTPStatus of %d, got %d", i, ss.Results[i].HTTPStatus, expectedResult.HTTPStatus)
				}
				if expectedResult.DNSRCode != ss.Results[i].DNSRCode {
					t.Errorf("Result at index %d should've had a DNSRCode of %s, got %s", i, ss.Results[i].DNSRCode, expectedResult.DNSRCode)
				}
				if expectedResult.Hostname != ss.Results[i].Hostname {
					t.Errorf("Result at index %d should've had a Hostname of %s, got %s", i, ss.Results[i].Hostname, expectedResult.Hostname)
				}
				if expectedResult.IP != ss.Results[i].IP {
					t.Errorf("Result at index %d should've had a IP of %s, got %s", i, ss.Results[i].IP, expectedResult.IP)
				}
				if expectedResult.Connected != ss.Results[i].Connected {
					t.Errorf("Result at index %d should've had a Connected value of %t, got %t", i, ss.Results[i].Connected, expectedResult.Connected)
				}
				if expectedResult.Duration != ss.Results[i].Duration {
					t.Errorf("Result at index %d should've had a Duration of %s, got %s", i, ss.Results[i].Duration.String(), expectedResult.Duration.String())
				}
				if len(expectedResult.Errors) != len(ss.Results[i].Errors) {
					t.Errorf("Result at index %d should've had %d errors, but actually had %d errors", i, len(ss.Results[i].Errors), len(expectedResult.Errors))
				} else {
					for j := range expectedResult.Errors {
						if ss.Results[i].Errors[j] != expectedResult.Errors[j] {
							t.Error("should've been the same")
						}
					}
				}
				if len(expectedResult.ConditionResults) != len(ss.Results[i].ConditionResults) {
					t.Errorf("Result at index %d should've had %d ConditionResults, but actually had %d ConditionResults", i, len(ss.Results[i].ConditionResults), len(expectedResult.ConditionResults))
				} else {
					for j := range expectedResult.ConditionResults {
						if ss.Results[i].ConditionResults[j].Condition != expectedResult.ConditionResults[j].Condition {
							t.Error("should've been the same")
						}
						if ss.Results[i].ConditionResults[j].Success != expectedResult.ConditionResults[j].Success {
							t.Error("should've been the same")
						}
					}
				}
				if expectedResult.Success != ss.Results[i].Success {
					t.Errorf("Result at index %d should've had a Success of %t, got %t", i, ss.Results[i].Success, expectedResult.Success)
				}
				if expectedResult.Timestamp.Unix() != ss.Results[i].Timestamp.Unix() {
					t.Errorf("Result at index %d should've had a Timestamp of %d, got %d", i, ss.Results[i].Timestamp.Unix(), expectedResult.Timestamp.Unix())
				}
				if expectedResult.CertificateExpiration != ss.Results[i].CertificateExpiration {
					t.Errorf("Result at index %d should've had a CertificateExpiration of %s, got %s", i, ss.Results[i].CertificateExpiration.String(), expectedResult.CertificateExpiration.String())
				}
			}
		})
	}
}

func TestStore_DeleteAllServiceStatusesNotInKeys(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_DeleteAllServiceStatusesNotInKeys")
	defer cleanUp(scenarios)
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
			// Delete everything
			scenario.Store.DeleteAllServiceStatusesNotInKeys([]string{})
			if len(scenario.Store.GetAllServiceStatuses(paging.NewServiceStatusParams())) != 0 {
				t.Errorf("everything should've been deleted")
			}
		})
	}
}

package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/gatus/v5/storage/store/memory"
	"github.com/TwiN/gatus/v5/storage/store/sql"
)

var (
	firstCondition  = endpoint.Condition("[STATUS] == 200")
	secondCondition = endpoint.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = endpoint.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	now = time.Now().Truncate(time.Hour)

	testEndpoint = endpoint.Endpoint{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []endpoint.Condition{firstCondition, secondCondition, thirdCondition},
		Alerts:                  nil,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuccessfulResult = endpoint.Result{
		Timestamp:             now,
		Success:               true,
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                nil,
		Connected:             true,
		Duration:              150 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*endpoint.ConditionResult{
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
	testUnsuccessfulResult = endpoint.Result{
		Timestamp:             now,
		Success:               false,
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                []string{"error-1", "error-2"},
		Connected:             true,
		Duration:              750 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*endpoint.ConditionResult{
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
	memoryStore, err := memory.NewStore(storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sql.NewStore("sqlite", t.TempDir()+"/"+testName+".db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("failed to create store:", err.Error())
	}
	sqliteStoreWithCaching, err := sql.NewStore("sqlite", t.TempDir()+"/"+testName+"-with-caching.db", true, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
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
		{
			Name:  "sqlite-with-caching",
			Store: sqliteStoreWithCaching,
		},
	}
}

func cleanUp(scenarios []*Scenario) {
	for _, scenario := range scenarios {
		scenario.Store.Close()
	}
}

func TestStore_GetEndpointStatusByKey(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetEndpointStatusByKey")
	defer cleanUp(scenarios)
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-time.Minute)
	secondResult := testUnsuccessfulResult
	secondResult.Timestamp = now
	thirdResult := testSuccessfulResult
	thirdResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.InsertEndpointResult(&testEndpoint, &firstResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &secondResult)
			endpointStatus, err := scenario.Store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithEvents(1, storage.DefaultMaximumNumberOfEvents).WithResults(1, storage.DefaultMaximumNumberOfResults))
			if err != nil {
				t.Fatal("shouldn't have returned an error, got", err.Error())
			}
			if endpointStatus == nil {
				t.Fatalf("endpointStatus shouldn't have been nil")
			}
			if endpointStatus.Name != testEndpoint.Name {
				t.Fatalf("endpointStatus.Name should've been %s, got %s", testEndpoint.Name, endpointStatus.Name)
			}
			if endpointStatus.Group != testEndpoint.Group {
				t.Fatalf("endpointStatus.Group should've been %s, got %s", testEndpoint.Group, endpointStatus.Group)
			}
			if len(endpointStatus.Results) != 2 {
				t.Fatalf("endpointStatus.Results should've had 2 entries")
			}
			if endpointStatus.Results[0].Timestamp.After(endpointStatus.Results[1].Timestamp) {
				t.Error("The result at index 0 should've been older than the result at index 1")
			}
			scenario.Store.InsertEndpointResult(&testEndpoint, &thirdResult)
			endpointStatus, err = scenario.Store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithEvents(1, storage.DefaultMaximumNumberOfEvents).WithResults(1, storage.DefaultMaximumNumberOfResults))
			if err != nil {
				t.Fatal("shouldn't have returned an error, got", err.Error())
			}
			if len(endpointStatus.Results) != 3 {
				t.Fatalf("endpointStatus.Results should've had 3 entries")
			}
			scenario.Store.Clear()
		})
	}
}

func TestStore_GetEndpointStatusForMissingStatusReturnsNil(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetEndpointStatusForMissingStatusReturnsNil")
	defer cleanUp(scenarios)
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
			endpointStatus, err := scenario.Store.GetEndpointStatus("nonexistantgroup", "nonexistantname", paging.NewEndpointStatusParams().WithEvents(1, storage.DefaultMaximumNumberOfEvents).WithResults(1, storage.DefaultMaximumNumberOfResults))
			if !errors.Is(err, common.ErrEndpointNotFound) {
				t.Error("should've returned ErrEndpointNotFound, got", err)
			}
			if endpointStatus != nil {
				t.Errorf("Returned endpoint status for group '%s' and name '%s' not nil after inserting the endpoint into the store", testEndpoint.Group, testEndpoint.Name)
			}
			endpointStatus, err = scenario.Store.GetEndpointStatus(testEndpoint.Group, "nonexistantname", paging.NewEndpointStatusParams().WithEvents(1, storage.DefaultMaximumNumberOfEvents).WithResults(1, storage.DefaultMaximumNumberOfResults))
			if !errors.Is(err, common.ErrEndpointNotFound) {
				t.Error("should've returned ErrEndpointNotFound, got", err)
			}
			if endpointStatus != nil {
				t.Errorf("Returned endpoint status for group '%s' and name '%s' not nil after inserting the endpoint into the store", testEndpoint.Group, "nonexistantname")
			}
			endpointStatus, err = scenario.Store.GetEndpointStatus("nonexistantgroup", testEndpoint.Name, paging.NewEndpointStatusParams().WithEvents(1, storage.DefaultMaximumNumberOfEvents).WithResults(1, storage.DefaultMaximumNumberOfResults))
			if !errors.Is(err, common.ErrEndpointNotFound) {
				t.Error("should've returned ErrEndpointNotFound, got", err)
			}
			if endpointStatus != nil {
				t.Errorf("Returned endpoint status for group '%s' and name '%s' not nil after inserting the endpoint into the store", "nonexistantgroup", testEndpoint.Name)
			}
		})
	}
}

func TestStore_GetAllEndpointStatuses(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetAllEndpointStatuses")
	defer cleanUp(scenarios)
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult)
			endpointStatuses, err := scenario.Store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20))
			if err != nil {
				t.Error("shouldn't have returned an error, got", err.Error())
			}
			if len(endpointStatuses) != 1 {
				t.Fatal("expected 1 endpoint status")
			}
			actual := endpointStatuses[0]
			if actual == nil {
				t.Fatal("expected endpoint status to exist")
			}
			if len(actual.Results) != 2 {
				t.Error("expected 2 results, got", len(actual.Results))
			}
			if len(actual.Events) != 0 {
				t.Error("expected 0 events, got", len(actual.Events))
			}
			scenario.Store.Clear()
		})
		t.Run(scenario.Name+"-page-2", func(t *testing.T) {
			otherEndpoint := testEndpoint
			otherEndpoint.Name = testEndpoint.Name + "-other"
			scenario.Store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
			scenario.Store.InsertEndpointResult(&otherEndpoint, &testSuccessfulResult)
			scenario.Store.InsertEndpointResult(&otherEndpoint, &testSuccessfulResult)
			scenario.Store.InsertEndpointResult(&otherEndpoint, &testSuccessfulResult)
			endpointStatuses, err := scenario.Store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(2, 2))
			if err != nil {
				t.Error("shouldn't have returned an error, got", err.Error())
			}
			if len(endpointStatuses) != 2 {
				t.Fatal("expected 2 endpoint statuses")
			}
			if endpointStatuses[0] == nil || endpointStatuses[1] == nil {
				t.Fatal("expected endpoint status to exist")
			}
			if len(endpointStatuses[0].Results) != 0 {
				t.Error("expected 0 results on the first endpoint, got", len(endpointStatuses[0].Results))
			}
			if len(endpointStatuses[1].Results) != 1 {
				t.Error("expected 1 result on the second endpoint, got", len(endpointStatuses[1].Results))
			}
			if len(endpointStatuses[0].Events) != 0 {
				t.Error("expected 0 events on the first endpoint, got", len(endpointStatuses[0].Events))
			}
			if len(endpointStatuses[1].Events) != 0 {
				t.Error("expected 0 events on the second endpoint, got", len(endpointStatuses[1].Events))
			}
			scenario.Store.Clear()
		})
	}
}

func TestStore_GetAllEndpointStatusesWithResultsAndEvents(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetAllEndpointStatusesWithResultsAndEvents")
	defer cleanUp(scenarios)
	firstResult := testSuccessfulResult
	secondResult := testUnsuccessfulResult
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.InsertEndpointResult(&testEndpoint, &firstResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &secondResult)
			// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
			endpointStatuses, err := scenario.Store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20).WithEvents(1, 50))
			if err != nil {
				t.Error("shouldn't have returned an error, got", err.Error())
			}
			if len(endpointStatuses) != 1 {
				t.Fatal("expected 1 endpoint status")
			}
			actual := endpointStatuses[0]
			if actual == nil {
				t.Fatal("expected endpoint status to exist")
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

func TestStore_GetEndpointStatusPage1IsHasMoreRecentResultsThanPage2(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_GetEndpointStatusPage1IsHasMoreRecentResultsThanPage2")
	defer cleanUp(scenarios)
	firstResult := testSuccessfulResult
	firstResult.Timestamp = now.Add(-time.Minute)
	secondResult := testUnsuccessfulResult
	secondResult.Timestamp = now
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.InsertEndpointResult(&testEndpoint, &firstResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &secondResult)
			endpointStatusPage1, err := scenario.Store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithResults(1, 1))
			if err != nil {
				t.Error("shouldn't have returned an error, got", err.Error())
			}
			if endpointStatusPage1 == nil {
				t.Fatalf("endpointStatusPage1 shouldn't have been nil")
			}
			if len(endpointStatusPage1.Results) != 1 {
				t.Fatalf("endpointStatusPage1 should've had 1 result")
			}
			endpointStatusPage2, err := scenario.Store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithResults(2, 1))
			if err != nil {
				t.Error("shouldn't have returned an error, got", err.Error())
			}
			if endpointStatusPage2 == nil {
				t.Fatalf("endpointStatusPage2 shouldn't have been nil")
			}
			if len(endpointStatusPage2.Results) != 1 {
				t.Fatalf("endpointStatusPage2 should've had 1 result")
			}
			// Compare the timestamp of both pages
			if !endpointStatusPage1.Results[0].Timestamp.After(endpointStatusPage2.Results[0].Timestamp) {
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
			if _, err := scenario.Store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); err != common.ErrEndpointNotFound {
				t.Errorf("should've returned not found because there's nothing yet, got %v", err)
			}
			scenario.Store.InsertEndpointResult(&testEndpoint, &firstResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &secondResult)
			if uptime, _ := scenario.Store.GetUptimeByKey(testEndpoint.Key(), now.Add(-time.Hour), time.Now()); uptime != 0.5 {
				t.Errorf("the uptime over the past 1h should've been 0.5, got %f", uptime)
			}
			if uptime, _ := scenario.Store.GetUptimeByKey(testEndpoint.Key(), now.Add(-time.Hour*24), time.Now()); uptime != 0.5 {
				t.Errorf("the uptime over the past 24h should've been 0.5, got %f", uptime)
			}
			if uptime, _ := scenario.Store.GetUptimeByKey(testEndpoint.Key(), now.Add(-time.Hour*24*7), time.Now()); uptime != 0.5 {
				t.Errorf("the uptime over the past 7d should've been 0.5, got %f", uptime)
			}
			if _, err := scenario.Store.GetUptimeByKey(testEndpoint.Key(), now, time.Now().Add(-time.Hour)); err == nil {
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
			scenario.Store.InsertEndpointResult(&testEndpoint, &firstResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &secondResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &thirdResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &fourthResult)
			if averageResponseTime, err := scenario.Store.GetAverageResponseTimeByKey(testEndpoint.Key(), now.Add(-48*time.Hour), now.Add(-24*time.Hour)); err == nil {
				if averageResponseTime != 0 {
					t.Errorf("expected average response time to be 0ms, got %v", averageResponseTime)
				}
			} else {
				t.Error("shouldn't have returned an error, got", err)
			}
			if averageResponseTime, err := scenario.Store.GetAverageResponseTimeByKey(testEndpoint.Key(), now.Add(-24*time.Hour), now); err == nil {
				if averageResponseTime != 287 {
					t.Errorf("expected average response time to be 287ms, got %v", averageResponseTime)
				}
			} else {
				t.Error("shouldn't have returned an error, got", err)
			}
			if averageResponseTime, err := scenario.Store.GetAverageResponseTimeByKey(testEndpoint.Key(), now.Add(-time.Hour), now); err == nil {
				if averageResponseTime != 350 {
					t.Errorf("expected average response time to be 350ms, got %v", averageResponseTime)
				}
			} else {
				t.Error("shouldn't have returned an error, got", err)
			}
			if averageResponseTime, err := scenario.Store.GetAverageResponseTimeByKey(testEndpoint.Key(), now.Add(-2*time.Hour), now.Add(-time.Hour)); err == nil {
				if averageResponseTime != 216 {
					t.Errorf("expected average response time to be 216ms, got %v", averageResponseTime)
				}
			} else {
				t.Error("shouldn't have returned an error, got", err)
			}
			if _, err := scenario.Store.GetAverageResponseTimeByKey(testEndpoint.Key(), now, now.Add(-2*time.Hour)); err == nil {
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
			scenario.Store.InsertEndpointResult(&testEndpoint, &firstResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &secondResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &thirdResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &fourthResult)
			hourlyAverageResponseTime, err := scenario.Store.GetHourlyAverageResponseTimeByKey(testEndpoint.Key(), now.Add(-24*time.Hour), now)
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
			scenario.Store.InsertEndpointResult(&testEndpoint, &firstResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &secondResult)
			ss, err := scenario.Store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithEvents(1, storage.DefaultMaximumNumberOfEvents).WithResults(1, storage.DefaultMaximumNumberOfResults))
			if err != nil {
				t.Error("shouldn't have returned an error, got", err)
			}
			if ss == nil {
				t.Fatalf("Store should've had key '%s', but didn't", testEndpoint.Key())
			}
			if len(ss.Events) != 3 {
				t.Fatalf("Endpoint '%s' should've had 3 events, got %d", ss.Name, len(ss.Events))
			}
			if len(ss.Results) != 2 {
				t.Fatalf("Endpoint '%s' should've had 2 results, got %d", ss.Name, len(ss.Results))
			}
			for i, expectedResult := range []endpoint.Result{firstResult, secondResult} {
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

func TestStore_DeleteAllEndpointStatusesNotInKeys(t *testing.T) {
	scenarios := initStoresAndBaseScenarios(t, "TestStore_DeleteAllEndpointStatusesNotInKeys")
	defer cleanUp(scenarios)
	firstEndpoint := endpoint.Endpoint{Name: "endpoint-1", Group: "group"}
	secondEndpoint := endpoint.Endpoint{Name: "endpoint-2", Group: "group"}
	r := &testSuccessfulResult
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Store.InsertEndpointResult(&firstEndpoint, r)
			scenario.Store.InsertEndpointResult(&secondEndpoint, r)
			if ss, _ := scenario.Store.GetEndpointStatusByKey(firstEndpoint.Key(), paging.NewEndpointStatusParams()); ss == nil {
				t.Fatal("firstEndpoint should exist, got", ss)
			}
			if ss, _ := scenario.Store.GetEndpointStatusByKey(secondEndpoint.Key(), paging.NewEndpointStatusParams()); ss == nil {
				t.Fatal("secondEndpoint should exist, got", ss)
			}
			scenario.Store.DeleteAllEndpointStatusesNotInKeys([]string{firstEndpoint.Key()})
			if ss, _ := scenario.Store.GetEndpointStatusByKey(firstEndpoint.Key(), paging.NewEndpointStatusParams()); ss == nil {
				t.Error("secondEndpoint should still exist, got", ss)
			}
			if ss, _ := scenario.Store.GetEndpointStatusByKey(secondEndpoint.Key(), paging.NewEndpointStatusParams()); ss != nil {
				t.Error("firstEndpoint should have been deleted, got", ss)
			}
			// Delete everything
			scenario.Store.DeleteAllEndpointStatusesNotInKeys([]string{})
			endpointStatuses, _ := scenario.Store.GetAllEndpointStatuses(paging.NewEndpointStatusParams())
			if len(endpointStatuses) != 0 {
				t.Errorf("everything should've been deleted")
			}
		})
	}
}

func TestGet(t *testing.T) {
	store := Get()
	if store == nil {
		t.Error("store should've been automatically initialized")
	}
}

func TestInitialize(t *testing.T) {
	dir := t.TempDir()
	type Scenario struct {
		Name        string
		Cfg         *storage.Config
		ExpectedErr error
	}
	scenarios := []Scenario{
		{
			Name:        "nil",
			Cfg:         nil,
			ExpectedErr: nil,
		},
		{
			Name:        "blank",
			Cfg:         &storage.Config{},
			ExpectedErr: nil,
		},
		{
			Name:        "memory-no-path",
			Cfg:         &storage.Config{Type: storage.TypeMemory},
			ExpectedErr: nil,
		},
		{
			Name:        "sqlite-no-path",
			Cfg:         &storage.Config{Type: storage.TypeSQLite},
			ExpectedErr: sql.ErrPathNotSpecified,
		},
		{
			Name:        "sqlite-with-path",
			Cfg:         &storage.Config{Type: storage.TypeSQLite, Path: filepath.Join(dir, "TestInitialize_sqlite-with-path.db")},
			ExpectedErr: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := Initialize(scenario.Cfg)
			if err != scenario.ExpectedErr {
				t.Errorf("expected %v, got %v", scenario.ExpectedErr, err)
			}
			if err != nil {
				return
			}
			if cancelFunc == nil {
				t.Error("cancelFunc shouldn't have been nil")
			}
			if ctx == nil {
				t.Error("ctx shouldn't have been nil")
			}
			if store == nil {
				t.Fatal("provider shouldn't have been nit")
			}
			store.Close()
			// Try to initialize it again
			err = Initialize(scenario.Cfg)
			if !errors.Is(err, scenario.ExpectedErr) {
				t.Errorf("expected %v, got %v", scenario.ExpectedErr, err)
				return
			}
			store.Close()
		})
	}
}

func TestAutoSave(t *testing.T) {
	file := filepath.Join(t.TempDir(), "/TestAutoSave.db")
	if err := Initialize(&storage.Config{Path: file}); err != nil {
		t.Fatal("shouldn't have returned an error")
	}
	go autoSave(ctx, store, 3*time.Millisecond)
	time.Sleep(15 * time.Millisecond)
	cancelFunc()
	time.Sleep(50 * time.Millisecond)
}

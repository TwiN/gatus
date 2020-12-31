package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
)

var testService = core.Service{
	Name:                    "Name",
	Group:                   "Group",
	URL:                     "URL",
	DNS:                     &core.DNS{QueryType: "QueryType", QueryName: "QueryName"},
	Method:                  "Method",
	Body:                    "Body",
	GraphQL:                 false,
	Headers:                 nil,
	Interval:                time.Second * 2,
	Conditions:              nil,
	Alerts:                  nil,
	Insecure:                false,
	NumberOfFailuresInARow:  0,
	NumberOfSuccessesInARow: 0,
}

var memoryStore = NewInMemoryStore()

func TestStorage_GetAllFromEmptyMemoryStoreReturnsNothing(t *testing.T) {
	memoryStore.Clear()
	results := memoryStore.GetAll()
	if len(results) != 0 {
		t.Errorf("MemoryStore should've returned 0 results, but actually returned %d", len(results))
	}
}

func TestStorage_InsertIntoEmptyMemoryStoreThenGetAllReturnsOneResult(t *testing.T) {
	memoryStore.Clear()
	result := core.Result{
		HTTPStatus:            200,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "Hostname",
		IP:                    "IP",
		Connected:             false,
		Duration:              time.Second * 2,
		Errors:                nil,
		ConditionResults:      nil,
		Success:               false,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}

	memoryStore.Insert(&testService, &result)

	results := memoryStore.GetAll()
	if len(results) != 1 {
		t.Errorf("MemoryStore should've returned 0 results, but actually returned %d", len(results))
	}

	key := fmt.Sprintf("%s_%s", testService.Group, testService.Name)
	storedResult, exists := results[key]
	if !exists {
		t.Fatalf("In Memory Store should've contained key '%s', but didn't", key)
	}

	if storedResult.Name != testService.Name {
		t.Errorf("Stored Results Name should've been %s, but was %s", testService.Name, storedResult.Name)
	}
	if storedResult.Group != testService.Group {
		t.Errorf("Stored Results Group should've been %s, but was %s", testService.Group, storedResult.Group)
	}
	if len(storedResult.Results) != 1 {
		t.Errorf("Stored Results for service %s should've had 1 result, but actually had %d", storedResult.Name, len(storedResult.Results))
	}
	if storedResult.Results[0] == &result {
		t.Errorf("Returned result is the same reference as result passed to insert. Returned result should be copies only")
	}
}

func TestStorage_InsertTwoResultsForSingleServiceIntoEmptyMemoryStore_ThenGetAllReturnsTwoResults(t *testing.T) {
	memoryStore.Clear()
	result1 := core.Result{
		HTTPStatus:            404,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "Hostname",
		IP:                    "IP",
		Connected:             false,
		Duration:              time.Second * 2,
		Errors:                nil,
		ConditionResults:      nil,
		Success:               false,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}
	result2 := core.Result{
		HTTPStatus:            200,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "Hostname",
		IP:                    "IP",
		Connected:             true,
		Duration:              time.Second * 2,
		Errors:                nil,
		ConditionResults:      nil,
		Success:               true,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}

	resultsToInsert := []core.Result{result1, result2}

	memoryStore.Insert(&testService, &result1)
	memoryStore.Insert(&testService, &result2)

	results := memoryStore.GetAll()
	if len(results) != 1 {
		t.Fatalf("MemoryStore should've returned 1 results, but actually returned %d", len(results))
	}

	key := fmt.Sprintf("%s_%s", testService.Group, testService.Name)
	serviceResults, exists := results[key]
	if !exists {
		t.Fatalf("In Memory Store should've contained key '%s', but didn't", key)
	}

	if len(serviceResults.Results) != 2 {
		t.Fatalf("Service '%s' should've had 2 results, but actually returned %d", serviceResults.Name, len(serviceResults.Results))
	}

	for i, r := range serviceResults.Results {
		expectedResult := resultsToInsert[i]

		if r.HTTPStatus != expectedResult.HTTPStatus {
			t.Errorf("Result at index %d should've had a HTTPStatus of %d, but was actually %d", i, expectedResult.HTTPStatus, r.HTTPStatus)
		}
		if r.DNSRCode != expectedResult.DNSRCode {
			t.Errorf("Result at index %d should've had a DNSRCode of %s, but was actually %s", i, expectedResult.DNSRCode, r.DNSRCode)
		}
		if len(r.Body) != len(expectedResult.Body) {
			t.Errorf("Result at index %d should've had a body of length %d, but was actually %d", i, len(expectedResult.Body), len(r.Body))
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
}

func TestStorage_InsertTwoResultsTwoServicesIntoEmptyMemoryStore_ThenGetAllReturnsTwoServicesWithOneResultEach(t *testing.T) {
	memoryStore.Clear()
	result1 := core.Result{
		HTTPStatus:            404,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "Hostname",
		IP:                    "IP",
		Connected:             false,
		Duration:              time.Second * 2,
		Errors:                nil,
		ConditionResults:      nil,
		Success:               false,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}
	result2 := core.Result{
		HTTPStatus:            200,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "Hostname",
		IP:                    "IP",
		Connected:             true,
		Duration:              time.Second * 2,
		Errors:                nil,
		ConditionResults:      nil,
		Success:               true,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}

	testService2 := core.Service{
		Name:                    "Name2",
		Group:                   "Group",
		URL:                     "URL",
		DNS:                     &core.DNS{QueryType: "QueryType", QueryName: "QueryName"},
		Method:                  "Method",
		Body:                    "Body",
		GraphQL:                 false,
		Headers:                 nil,
		Interval:                time.Second * 2,
		Conditions:              nil,
		Alerts:                  nil,
		Insecure:                false,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}

	memoryStore.Insert(&testService, &result1)
	memoryStore.Insert(&testService2, &result2)

	results := memoryStore.GetAll()
	if len(results) != 2 {
		t.Fatalf("MemoryStore should've returned 2 results, but actually returned %d", len(results))
	}

	key := fmt.Sprintf("%s_%s", testService.Group, testService.Name)
	serviceResults1, exists := results[key]
	if !exists {
		t.Fatalf("In Memory Store should've contained key '%s', but didn't", key)
	}

	if len(serviceResults1.Results) != 1 {
		t.Fatalf("Service '%s' should've had 1 results, but actually returned %d", serviceResults1.Name, len(serviceResults1.Results))
	}

	key = fmt.Sprintf("%s_%s", testService2.Group, testService2.Name)
	serviceResults2, exists := results[key]
	if !exists {
		t.Fatalf("In Memory Store should've contained key '%s', but didn't", key)
	}

	if len(serviceResults2.Results) != 1 {
		t.Fatalf("Service '%s' should've had 1 results, but actually returned %d", serviceResults1.Name, len(serviceResults1.Results))
	}
}

func TestStorage_InsertResultForServiceWithErrorsIntoEmptyMemoryStore_ThenGetAllReturnsOneResultWithErrors(t *testing.T) {
	memoryStore.Clear()
	errors := []string{
		"error1",
		"error2",
	}
	result1 := core.Result{
		HTTPStatus:            404,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "Hostname",
		IP:                    "IP",
		Connected:             false,
		Duration:              time.Second * 2,
		Errors:                errors,
		ConditionResults:      nil,
		Success:               false,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}

	memoryStore.Insert(&testService, &result1)

	results := memoryStore.GetAll()
	if len(results) != 1 {
		t.Fatalf("MemoryStore should've returned 1 results, but actually returned %d", len(results))
	}

	key := fmt.Sprintf("%s_%s", testService.Group, testService.Name)
	serviceResults, exists := results[key]
	if !exists {
		t.Fatalf("In Memory Store should've contained key '%s', but didn't", key)
	}

	if len(serviceResults.Results) != 1 {
		t.Fatalf("Service '%s' should've had 2 results, but actually returned %d", serviceResults.Name, len(serviceResults.Results))
	}

	actualResult := serviceResults.Results[0]

	if len(actualResult.Errors) != len(errors) {
		t.Errorf("Service result should've had 2 errors, but actually had %d errors", len(actualResult.Errors))
	}

	for i, err := range actualResult.Errors {
		if err != errors[i] {
			t.Errorf("Error at index %d should've been %s, but was actually %s", i, errors[i], err)
		}
	}
}

func TestStorage_InsertResultForServiceWithConditionResultsIntoEmptyMemoryStore_ThenGetAllReturnsOneResultWithConditionResults(t *testing.T) {
	memoryStore.Clear()
	crs := []*core.ConditionResult{
		{
			Condition: "condition1",
			Success:   true,
		},
		{
			Condition: "condition2",
			Success:   false,
		},
	}
	result := core.Result{
		HTTPStatus:            404,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "Hostname",
		IP:                    "IP",
		Connected:             false,
		Duration:              time.Second * 2,
		Errors:                nil,
		ConditionResults:      crs,
		Success:               false,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}

	memoryStore.Insert(&testService, &result)

	results := memoryStore.GetAll()
	if len(results) != 1 {
		t.Fatalf("MemoryStore should've returned 1 results, but actually returned %d", len(results))
	}

	key := fmt.Sprintf("%s_%s", testService.Group, testService.Name)
	serviceResults, exists := results[key]
	if !exists {
		t.Fatalf("In Memory Store should've contained key '%s', but didn't", key)
	}

	if len(serviceResults.Results) != 1 {
		t.Fatalf("Service '%s' should've had 2 results, but actually returned %d", serviceResults.Name, len(serviceResults.Results))
	}

	actualResult := serviceResults.Results[0]

	if len(actualResult.ConditionResults) != len(crs) {
		t.Errorf("Service result should've had 2 ConditionResults, but actually had %d ConditionResults", len(actualResult.Errors))
	}

	for i, cr := range actualResult.ConditionResults {
		if cr.Condition != crs[i].Condition {
			t.Errorf("ConditionResult at index %d should've had condition %s, but was actually %s", i, crs[i].Condition, cr.Condition)
		}
		if cr.Success != crs[i].Success {
			t.Errorf("ConditionResult at index %d should've had success value of %t, but was actually %t", i, crs[i].Success, cr.Success)
		}
	}
}

func TestStorage_MultipleMemoryStoreInstancesReferToDifferentInternalMaps(t *testing.T) {
	memoryStore.Clear()
	currentMap := memoryStore.GetAll()

	otherMemoryStore := NewInMemoryStore()
	otherMemoryStoresMap := otherMemoryStore.GetAll()

	if len(currentMap) != len(otherMemoryStoresMap) {
		t.Errorf("Multiple memory stores should refer to the different internal maps, but 'memoryStore' returned %d results, and 'otherMemoryStore' returned %d results", len(currentMap), len(otherMemoryStoresMap))
	}

	memoryStore.Insert(&testService, &core.Result{})
	currentMap = memoryStore.GetAll()
	otherMemoryStoresMap = otherMemoryStore.GetAll()

	if len(currentMap) == len(otherMemoryStoresMap) {
		t.Errorf("Multiple memory stores should refer to different internal maps, but 'memoryStore' returned %d results after inserting, and 'otherMemoryStore' returned %d results after inserting", len(currentMap), len(otherMemoryStoresMap))
	}

	otherMemoryStore.Clear()
	currentMap = memoryStore.GetAll()
	otherMemoryStoresMap = otherMemoryStore.GetAll()

	if len(currentMap) == len(otherMemoryStoresMap) {
		t.Errorf("Multiple memory stores should refer to different internal maps, but 'memoryStore' returned %d results after clearing, and 'otherMemoryStore' returned %d results after clearing", len(currentMap), len(otherMemoryStoresMap))
	}
}

func TestStorage_ModificationsToReturnedMapDoNotAffectInternalMap(t *testing.T) {
	memoryStore.Clear()

	memoryStore.Insert(&testService, &core.Result{})
	modifiedResults := memoryStore.GetAll()
	for k := range modifiedResults {
		delete(modifiedResults, k)
	}
	results := memoryStore.GetAll()

	if len(modifiedResults) == len(results) {
		t.Errorf("Returned map from GetAll should be free to modify by the caller without affecting internal in-memory map, but length of results from in-memory map (%d) was equal to the length of results in modified map (%d)", len(results), len(modifiedResults))
	}
}

func TestStorage_GetServiceStatusForExistingStatusReturnsThatServiceStatus(t *testing.T) {
	memoryStore.Clear()

	memoryStore.Insert(&testService, &core.Result{})
	serviceStatus := memoryStore.GetServiceStatus(testService.Group, testService.Name)

	if serviceStatus == nil {
		t.Errorf("Returned service status for group '%s' and name '%s' was nil after inserting the service into the store", testService.Group, testService.Name)
	}
}

func TestStorage_GetServiceStatusForMissingStatusReturnsNil(t *testing.T) {
	memoryStore.Clear()

	memoryStore.Insert(&testService, &core.Result{})

	serviceStatus := memoryStore.GetServiceStatus("nonexistantgroup", "nonexistantname")
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, testService.Name)
	}

	serviceStatus = memoryStore.GetServiceStatus(testService.Group, "nonexistantname")
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, "nonexistantname")
	}

	serviceStatus = memoryStore.GetServiceStatus("nonexistantgroup", testService.Name)
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", "nonexistantgroup", testService.Name)
	}
}

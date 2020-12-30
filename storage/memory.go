package storage

import (
	"fmt"
	"sync"

	"github.com/TwinProduction/gatus/core"
)

var (
	serviceStatuses = make(map[string]*core.ServiceStatus)

	// serviceResultsMutex is used to prevent concurrent map access
	serviceResultsMutex sync.RWMutex
)

// InMemoryStore implements an in-memory store
type InMemoryStore struct{}

// NewInMemoryStore returns an in-memory store. Note that the store acts as a singleton, so although new-ing
// up in-memory stores will give you a unique reference to a struct each time, all structs returned
// by this function will act on the same in-memory store.
func NewInMemoryStore() InMemoryStore {
	return InMemoryStore{}
}

// GetAll returns all the observed results for all services from the in memory store
func (ims *InMemoryStore) GetAll() map[string]*core.ServiceStatus {
	results := make(map[string]*core.ServiceStatus)
	serviceResultsMutex.RLock()
	for key, svcStatus := range serviceStatuses {
		copiedResults := copyResults(svcStatus.Results)
		results[key] = &core.ServiceStatus{
			Name:    svcStatus.Name,
			Group:   svcStatus.Group,
			Results: copiedResults,
		}
	}
	serviceResultsMutex.RUnlock()

	return results
}

// Insert inserts the observed result for the specified service into the in memory store
func (ims *InMemoryStore) Insert(service *core.Service, result *core.Result) {
	key := fmt.Sprintf("%s_%s", service.Group, service.Name)
	serviceResultsMutex.Lock()
	serviceStatus, exists := serviceStatuses[key]
	if !exists {
		serviceStatus = core.NewServiceStatus(service)
		serviceStatuses[key] = serviceStatus
	}
	serviceStatus.AddResult(result)
	serviceResultsMutex.Unlock()
}

func copyResults(results []*core.Result) []*core.Result {
	copiedResults := []*core.Result{}
	for _, result := range results {
		copiedErrors := copyErrors(result.Errors)
		copiedConditionResults := copyConditionResults(result.ConditionResults)

		copiedResults = append(copiedResults, &core.Result{
			HTTPStatus:            result.HTTPStatus,
			DNSRCode:              result.DNSRCode,
			Body:                  result.Body,
			Hostname:              result.Hostname,
			IP:                    result.IP,
			Connected:             result.Connected,
			Duration:              result.Duration,
			Errors:                copiedErrors,
			ConditionResults:      copiedConditionResults,
			Success:               result.Connected,
			Timestamp:             result.Timestamp,
			CertificateExpiration: result.CertificateExpiration,
		})
	}
	return copiedResults
}

func copyConditionResults(crs []*core.ConditionResult) []*core.ConditionResult {
	copiedConditionResults := []*core.ConditionResult{}
	for _, conditionResult := range crs {
		copiedConditionResults = append(copiedConditionResults, &core.ConditionResult{
			Condition: conditionResult.Condition,
			Success:   conditionResult.Success,
		})
	}

	return copiedConditionResults
}

func copyErrors(errors []string) []string {
	copiedErrors := []string{}
	for _, error := range errors {
		copiedErrors = append(copiedErrors, error)
	}
	return copiedErrors
}

// Clear will empty all the results from the in memory store
func (ims *InMemoryStore) Clear() {
	serviceResultsMutex.Lock()
	serviceStatuses = make(map[string]*core.ServiceStatus)
	serviceResultsMutex.Unlock()
}

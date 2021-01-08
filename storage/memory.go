package storage

import (
	"fmt"
	"sync"

	"github.com/TwinProduction/gatus/core"
)

// InMemoryStore implements an in-memory store
type InMemoryStore struct {
	serviceStatuses     map[string]*core.ServiceStatus
	serviceResultsMutex sync.RWMutex
}

// NewInMemoryStore returns an in-memory store. Note that the store acts as a singleton, so although new-ing
// up in-memory stores will give you a unique reference to a struct each time, all structs returned
// by this function will act on the same in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		serviceStatuses: make(map[string]*core.ServiceStatus),
	}
}

// GetAll returns all the observed results for all services from the in memory store
func (ims *InMemoryStore) GetAll() map[string]*core.ServiceStatus {
	results := make(map[string]*core.ServiceStatus, len(ims.serviceStatuses))
	ims.serviceResultsMutex.RLock()
	for key, serviceStatus := range ims.serviceStatuses {
		results[key] = &core.ServiceStatus{
			Name:    serviceStatus.Name,
			Group:   serviceStatus.Group,
			Results: copyResults(serviceStatus.Results),
		}
	}
	ims.serviceResultsMutex.RUnlock()
	return results
}

// GetServiceStatus returns the service status for a given service name in the given group
func (ims *InMemoryStore) GetServiceStatus(group, name string) *core.ServiceStatus {
	key := fmt.Sprintf("%s_%s", group, name)
	ims.serviceResultsMutex.RLock()
	serviceStatus := ims.serviceStatuses[key]
	ims.serviceResultsMutex.RUnlock()
	return serviceStatus
}

// Insert inserts the observed result for the specified service into the in memory store
func (ims *InMemoryStore) Insert(service *core.Service, result *core.Result) {
	key := fmt.Sprintf("%s_%s", service.Group, service.Name)
	ims.serviceResultsMutex.Lock()
	serviceStatus, exists := ims.serviceStatuses[key]
	if !exists {
		serviceStatus = core.NewServiceStatus(service)
		ims.serviceStatuses[key] = serviceStatus
	}
	serviceStatus.AddResult(result)
	ims.serviceResultsMutex.Unlock()
}

func copyResults(results []*core.Result) []*core.Result {
	var copiedResults []*core.Result
	for _, result := range results {
		copiedResults = append(copiedResults, &core.Result{
			HTTPStatus:            result.HTTPStatus,
			DNSRCode:              result.DNSRCode,
			Body:                  result.Body,
			Hostname:              result.Hostname,
			IP:                    result.IP,
			Connected:             result.Connected,
			Duration:              result.Duration,
			Errors:                copyErrors(result.Errors),
			ConditionResults:      copyConditionResults(result.ConditionResults),
			Success:               result.Success,
			Timestamp:             result.Timestamp,
			CertificateExpiration: result.CertificateExpiration,
		})
	}
	return copiedResults
}

func copyConditionResults(conditionResults []*core.ConditionResult) []*core.ConditionResult {
	var copiedConditionResults []*core.ConditionResult
	for _, conditionResult := range conditionResults {
		copiedConditionResults = append(copiedConditionResults, &core.ConditionResult{
			Condition: conditionResult.Condition,
			Success:   conditionResult.Success,
		})
	}
	return copiedConditionResults
}

func copyErrors(errors []string) []string {
	var copiedErrors []string
	for _, err := range errors {
		copiedErrors = append(copiedErrors, err)
	}
	return copiedErrors
}

// Clear will empty all the results from the in memory store
func (ims *InMemoryStore) Clear() {
	ims.serviceResultsMutex.Lock()
	ims.serviceStatuses = make(map[string]*core.ServiceStatus)
	ims.serviceResultsMutex.Unlock()
}

package memory

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/paging"
)

func TestAddResult(t *testing.T) {
	service := &core.Service{Name: "name", Group: "group"}
	serviceStatus := core.NewServiceStatus(service.Key(), service.Group, service.Name)
	for i := 0; i < core.MaximumNumberOfResults+10; i++ {
		AddResult(serviceStatus, &core.Result{Timestamp: time.Now()})
	}
	if len(serviceStatus.Results) != core.MaximumNumberOfResults {
		t.Errorf("expected serviceStatus.Results to not exceed a length of %d", core.MaximumNumberOfResults)
	}
}

func TestShallowCopyServiceStatus(t *testing.T) {
	service := &core.Service{Name: "name", Group: "group"}
	serviceStatus := core.NewServiceStatus(service.Key(), service.Group, service.Name)
	for i := 0; i < 25; i++ {
		AddResult(serviceStatus, &core.Result{Timestamp: time.Now()})
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(1, 1)).Results) != 1 {
		t.Errorf("expected to have 1 result")
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(5, 0)).Results) != 0 {
		t.Errorf("expected to have 0 results")
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(-1, 20)).Results) != 0 {
		t.Errorf("expected to have 0 result, because the page was invalid")
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(1, -1)).Results) != 0 {
		t.Errorf("expected to have 0 result, because the page size was invalid")
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(1, 10)).Results) != 10 {
		t.Errorf("expected to have 10 results, because given a page size of 10, page 1 should have 10 elements")
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(2, 10)).Results) != 10 {
		t.Errorf("expected to have 10 results, because given a page size of 10, page 2 should have 10 elements")
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(3, 10)).Results) != 5 {
		t.Errorf("expected to have 5 results, because given a page size of 10, page 3 should have 5 elements")
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(4, 10)).Results) != 0 {
		t.Errorf("expected to have 0 results, because given a page size of 10, page 4 should have 0 elements")
	}
	if len(ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(1, 50)).Results) != 25 {
		t.Errorf("expected to have 25 results, because there's only 25 results")
	}
}

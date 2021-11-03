package memory

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v3/core"
	"github.com/TwiN/gatus/v3/storage/store/common"
	"github.com/TwiN/gatus/v3/storage/store/common/paging"
)

func TestAddResult(t *testing.T) {
	endpoint := &core.Endpoint{Name: "name", Group: "group"}
	endpointStatus := core.NewEndpointStatus(endpoint.Group, endpoint.Name)
	for i := 0; i < (common.MaximumNumberOfResults+common.MaximumNumberOfEvents)*2; i++ {
		AddResult(endpointStatus, &core.Result{Success: i%2 == 0, Timestamp: time.Now()})
	}
	if len(endpointStatus.Results) != common.MaximumNumberOfResults {
		t.Errorf("expected endpointStatus.Results to not exceed a length of %d", common.MaximumNumberOfResults)
	}
	if len(endpointStatus.Events) != common.MaximumNumberOfEvents {
		t.Errorf("expected endpointStatus.Events to not exceed a length of %d", common.MaximumNumberOfEvents)
	}
	// Try to add nil endpointStatus
	AddResult(nil, &core.Result{Timestamp: time.Now()})
}

func TestShallowCopyEndpointStatus(t *testing.T) {
	endpoint := &core.Endpoint{Name: "name", Group: "group"}
	endpointStatus := core.NewEndpointStatus(endpoint.Group, endpoint.Name)
	ts := time.Now().Add(-25 * time.Hour)
	for i := 0; i < 25; i++ {
		AddResult(endpointStatus, &core.Result{Success: i%2 == 0, Timestamp: ts})
		ts = ts.Add(time.Hour)
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(-1, -1)).Results) != 0 {
		t.Error("expected to have 0 result")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(1, 1)).Results) != 1 {
		t.Error("expected to have 1 result")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(5, 0)).Results) != 0 {
		t.Error("expected to have 0 results")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(-1, 20)).Results) != 0 {
		t.Error("expected to have 0 result, because the page was invalid")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(1, -1)).Results) != 0 {
		t.Error("expected to have 0 result, because the page size was invalid")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(1, 10)).Results) != 10 {
		t.Error("expected to have 10 results, because given a page size of 10, page 1 should have 10 elements")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(2, 10)).Results) != 10 {
		t.Error("expected to have 10 results, because given a page size of 10, page 2 should have 10 elements")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(3, 10)).Results) != 5 {
		t.Error("expected to have 5 results, because given a page size of 10, page 3 should have 5 elements")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(4, 10)).Results) != 0 {
		t.Error("expected to have 0 results, because given a page size of 10, page 4 should have 0 elements")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(1, 50)).Results) != 25 {
		t.Error("expected to have 25 results, because there's only 25 results")
	}
}

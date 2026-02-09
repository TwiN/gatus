package memory

import (
	"testing"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

func BenchmarkShallowCopyEndpointStatus(b *testing.B) {
	ep := &testEndpoint
	status := endpoint.NewStatus(ep.Group, ep.Name)
	for range storage.DefaultMaximumNumberOfResults {
		AddResult(status, &testSuccessfulResult, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	}
	for b.Loop() {
		ShallowCopyEndpointStatus(status, paging.NewEndpointStatusParams().WithResults(1, 20))
	}
	b.ReportAllocs()
}

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
	for i := 0; i < storage.DefaultMaximumNumberOfResults; i++ {
		AddResult(status, &testSuccessfulResult, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	}
	for n := 0; n < b.N; n++ {
		ShallowCopyEndpointStatus(status, paging.NewEndpointStatusParams().WithResults(1, 20))
	}
	b.ReportAllocs()
}

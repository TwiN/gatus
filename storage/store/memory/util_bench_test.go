package memory

import (
	"github.com/TwiN/gatus/v5/storage"
	"testing"

	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

func BenchmarkShallowCopyEndpointStatus(b *testing.B) {
	endpoint := &testEndpoint
	status := core.NewEndpointStatus(endpoint.Group, endpoint.Name)
	for i := 0; i < storage.DefaultMaximumNumberOfResults; i++ {
		AddResult(status, &testSuccessfulResult, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	}
	for n := 0; n < b.N; n++ {
		ShallowCopyEndpointStatus(status, paging.NewEndpointStatusParams().WithResults(1, 20))
	}
	b.ReportAllocs()
}

package memory

import (
	"testing"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

func BenchmarkShallowCopyEndpointStatus(b *testing.B) {
	e := &testEndpoint
	status := endpoint.NewEndpointStatus(e.Group, e.Name)
	for i := 0; i < common.MaximumNumberOfResults; i++ {
		AddResult(status, &testSuccessfulResult)
	}
	for n := 0; n < b.N; n++ {
		ShallowCopyEndpointStatus(status, paging.NewEndpointStatusParams().WithResults(1, 20))
	}
	b.ReportAllocs()
}

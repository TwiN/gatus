package memory

import (
	"testing"

	"github.com/TwiN/gatus/v3/core"
	"github.com/TwiN/gatus/v3/storage/store/common"
	"github.com/TwiN/gatus/v3/storage/store/common/paging"
)

func BenchmarkShallowCopyServiceStatus(b *testing.B) {
	service := &testService
	serviceStatus := core.NewServiceStatus(service.Key(), service.Group, service.Name)
	for i := 0; i < common.MaximumNumberOfResults; i++ {
		AddResult(serviceStatus, &testSuccessfulResult)
	}
	for n := 0; n < b.N; n++ {
		ShallowCopyServiceStatus(serviceStatus, paging.NewServiceStatusParams().WithResults(1, 20))
	}
	b.ReportAllocs()
}

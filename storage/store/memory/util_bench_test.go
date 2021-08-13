package memory

import (
	"testing"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/common"
	"github.com/TwinProduction/gatus/storage/store/common/paging"
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

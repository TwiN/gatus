package endpoint

import (
	"testing"
)

func BenchmarkConvertGroupAndEndpointNameToKey(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ConvertGroupAndEndpointNameToKey("group", "name")
	}
}

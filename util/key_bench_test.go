package util

import (
	"testing"
)

func BenchmarkConvertGroupAndServiceToKey(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ConvertGroupAndServiceToKey("group", "service")
	}
}

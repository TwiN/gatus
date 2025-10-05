package key

import (
	"testing"
)

func BenchmarkConvertGroupAndNameToKey(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ConvertGroupAndNameToKey("group", "name")
	}
}
package endpoint

import (
	"testing"
)

func BenchmarkEndpoint_Type(b *testing.B) {
	for b.Loop() {
		for _, tt := range testEndpoint_typeData {
			endpoint := Endpoint{
				URL:       tt.args.URL,
				DNSConfig: tt.args.DNS,
			}
			if got := endpoint.Type(); got != tt.want {
				b.Errorf("Endpoint.Type() = %v, want %v", got, tt.want)
			}
		}
	}
}

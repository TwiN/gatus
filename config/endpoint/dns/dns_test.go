package dns

import (
	"testing"
)

func TestConfig_ValidateAndSetDefault(t *testing.T) {
	dns := &Config{
		QueryType: "A",
		QueryName: "",
	}
	err := dns.ValidateAndSetDefault()
	if err == nil {
		t.Error("Should've returned an error because endpoint's dns didn't have a query name, which is a mandatory field for dns")
	}
}

func TestConfig_ValidateAndSetDefaultsWithInvalidDNSQueryType(t *testing.T) {
	dns := &Config{
		QueryType: "B",
		QueryName: "example.com",
	}
	err := dns.ValidateAndSetDefault()
	if err == nil {
		t.Error("Should've returned an error because endpoint's dns query type is invalid, it needs to be a valid query name like A, AAAA, CNAME...")
	}
}

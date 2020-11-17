package core

import (
	"testing"
)

func TestIntegrationQuery(t *testing.T) {
	dns := DNS{
		QueryType: "A",
		QueryName: "example.com",
	}
	result := &Result{}
	dns.validateAndSetDefault()
	dns.query("8.8.8.8", result)
	if len(result.Errors) != 0 {
		t.Errorf("there should be no error Errors:%v", result.Errors)
	}

	if result.DNSRCode != "NOERROR" {
		t.Errorf("DNSRCode '%s' should have been NOERROR", result.DNSRCode)
	}

	if string(result.Body) != "93.184.216.34" {
		t.Errorf("expected result %s", "93.184.216.34")
	}
}

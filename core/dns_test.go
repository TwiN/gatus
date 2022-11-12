package core

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v4/pattern"
)

func TestIntegrationQuery(t *testing.T) {
	scenarios := []struct {
		name            string
		inputDNS        DNS
		inputURL        string
		expectedDNSCode string
		expectedBody    string
		isErrExpected   bool
	}{
		{
			name:            "dns-with-type-A",
			inputDNS:        DNS{QueryType: "A", QueryName: "example.com."},
			inputURL:        "1.1.1.1",
			expectedDNSCode: "NOERROR",
			expectedBody:    "93.184.216.34",
		},
		{
			name:            "dns-with-type-AAAA",
			inputDNS:        DNS{QueryType: "AAAA", QueryName: "example.com."},
			inputURL:        "1.1.1.1",
			expectedDNSCode: "NOERROR",
			expectedBody:    "2606:2800:220:1:248:1893:25c8:1946",
		},
		{
			name:            "dns-with-type-CNAME",
			inputDNS:        DNS{QueryType: "CNAME", QueryName: "en.wikipedia.org."},
			inputURL:        "1.1.1.1",
			expectedDNSCode: "NOERROR",
			expectedBody:    "dyna.wikimedia.org.",
		},
		{
			name:            "dns-with-type-MX",
			inputDNS:        DNS{QueryType: "MX", QueryName: "example.com."},
			inputURL:        "1.1.1.1",
			expectedDNSCode: "NOERROR",
			expectedBody:    ".",
		},
		{
			name:            "dns-with-type-NS",
			inputDNS:        DNS{QueryType: "NS", QueryName: "example.com."},
			inputURL:        "1.1.1.1",
			expectedDNSCode: "NOERROR",
			expectedBody:    "*.iana-servers.net.",
		},
		{
			name:          "dns-with-invalid-type",
			inputDNS:      DNS{QueryType: "B", QueryName: "example"},
			inputURL:      "1.1.1.1",
			isErrExpected: true,
		},
	}
	for _, test := range scenarios {
		t.Run(test.name, func(t *testing.T) {
			dns := test.inputDNS
			result := &Result{}
			dns.query(test.inputURL, result)
			if test.isErrExpected && len(result.Errors) == 0 {
				t.Errorf("there should be errors")
			}
			if result.DNSRCode != test.expectedDNSCode {
				t.Errorf("expected DNSRCode to be %s, got %s", test.expectedDNSCode, result.DNSRCode)
			}
			if test.inputDNS.QueryType == "NS" {
				// Because there are often multiple nameservers backing a single domain, we'll only look at the suffix
				if !pattern.Match(test.expectedBody, string(result.body)) {
					t.Errorf("expected [BODY] to be %s, got %s,", test.expectedBody, string(result.body))
				}
			} else {
				if string(result.body) != test.expectedBody {
					t.Errorf("expected [BODY] to be %s, got %s,", test.expectedBody, string(result.body))
				}
			}
		})
		time.Sleep(50 * time.Millisecond)
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithNoDNSQueryName(t *testing.T) {
	defer func() { recover() }()
	dns := &DNS{
		QueryType: "A",
		QueryName: "",
	}
	err := dns.validateAndSetDefault()
	if err == nil {
		t.Fatal("Should've returned an error because endpoint's dns didn't have a query name, which is a mandatory field for dns")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithInvalidDNSQueryType(t *testing.T) {
	defer func() { recover() }()
	dns := &DNS{
		QueryType: "B",
		QueryName: "example.com",
	}
	err := dns.validateAndSetDefault()
	if err == nil {
		t.Fatal("Should've returned an error because endpoint's dns query type is invalid, it needs to be a valid query name like A, AAAA, CNAME...")
	}
}

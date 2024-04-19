package core

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/pattern"
)

func TestIntegrationQuery(t *testing.T) {
	tests := []struct {
		name            string
		inputDNS        DNS
		inputURL        string
		expectedDNSCode string
		expectedBody    string
		isErrExpected   bool
	}{
		{
			name: "test DNS with type A",
			inputDNS: DNS{
				QueryType: "A",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "93.184.215.14",
		},
		{
			name: "test DNS with type AAAA",
			inputDNS: DNS{
				QueryType: "AAAA",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "2606:2800:21f:cb07:6820:80da:af6b:8b2c",
		},
		{
			name: "test DNS with type CNAME",
			inputDNS: DNS{
				QueryType: "CNAME",
				QueryName: "en.wikipedia.org.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "dyna.wikimedia.org.",
		},
		{
			name: "test DNS with type MX",
			inputDNS: DNS{
				QueryType: "MX",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    ".",
		},
		{
			name: "test DNS with type NS",
			inputDNS: DNS{
				QueryType: "NS",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "*.iana-servers.net.",
		},
		{
			name: "test DNS with fake type and retrieve error",
			inputDNS: DNS{
				QueryType: "B",
				QueryName: "example",
			},
			inputURL:      "8.8.8.8",
			isErrExpected: true,
		},
	}

	for _, test := range tests {
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
				if !pattern.Match(test.expectedBody, string(result.Body)) {
					t.Errorf("got %s, expected result %s,", string(result.Body), test.expectedBody)
				}
			} else {
				if string(result.Body) != test.expectedBody {
					t.Errorf("got %s, expected result %s,", string(result.Body), test.expectedBody)
				}
			}
		})
		time.Sleep(5 * time.Millisecond)
	}
}

func TestDNS_validateAndSetDefault(t *testing.T) {
	dns := &DNS{
		QueryType: "A",
		QueryName: "",
	}
	err := dns.validateAndSetDefault()
	if err == nil {
		t.Error("Should've returned an error because endpoint's dns didn't have a query name, which is a mandatory field for dns")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithInvalidDNSQueryType(t *testing.T) {
	dns := &DNS{
		QueryType: "B",
		QueryName: "example.com",
	}
	err := dns.validateAndSetDefault()
	if err == nil {
		t.Error("Should've returned an error because endpoint's dns query type is invalid, it needs to be a valid query name like A, AAAA, CNAME...")
	}
}

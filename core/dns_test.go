package core

import (
	"testing"

	"github.com/TwinProduction/gatus/pattern"
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
			expectedBody:    "93.184.216.34",
		},
		{
			name: "test DNS with type AAAA",
			inputDNS: DNS{
				QueryType: "AAAA",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "2606:2800:220:1:248:1893:25c8:1946",
		},
		{
			name: "test DNS with type CNAME",
			inputDNS: DNS{
				QueryType: "CNAME",
				QueryName: "doc.google.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "writely.l.google.com.",
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
				QueryName: "google",
			},
			inputURL:      "8.8.8.8",
			isErrExpected: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			dns := test.inputDNS
			result := &Result{}
			dns.query(test.inputURL, result)
			if test.isErrExpected && len(result.Errors) == 0 {
				t.Errorf("there should be errors")
			}
			if result.DNSRCode != test.expectedDNSCode {
				t.Errorf("DNSRCodePlaceholder '%s' should have been %s", result.DNSRCode, test.expectedDNSCode)
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
	}
}

func TestService_ValidateAndSetDefaultsWithNoDNSQueryName(t *testing.T) {
	defer func() { recover() }()
	dns := &DNS{
		QueryType: "A",
		QueryName: "",
	}
	dns.validateAndSetDefault()
	t.Fatal("Should've panicked because service`s dns didn't have a query name, which is a mandatory field for dns")
}

func TestService_ValidateAndSetDefaultsWithInvalidDNSQueryType(t *testing.T) {
	defer func() { recover() }()
	dns := &DNS{
		QueryType: "B",
		QueryName: "example.com",
	}
	dns.validateAndSetDefault()
	t.Fatal("Should've panicked because service`s dns query type is invalid, it needs to be a valid query name like A, AAAA, CNAME...")
}

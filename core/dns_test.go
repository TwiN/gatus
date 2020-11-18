package core

import (
	"testing"
)

func TestIntegrationQuery(t *testing.T) {
	tests := []struct {
		name            string
		inputDNS        DNS
		inputURL        string
		expectedDNSCode string
		expectedBody    string
	}{
		{
			name: "test DNS with type A",
			inputDNS: DNS{
				QueryType: "A",
				QueryName: "example.com",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "93.184.216.34",
		},
		{
			name: "test DNS with type AAAA",
			inputDNS: DNS{
				QueryType: "AAAA",
				QueryName: "example.com",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "2606:2800:220:1:248:1893:25c8:1946",
		},
		{
			name: "test DNS with type CNAME",
			inputDNS: DNS{
				QueryType: "CNAME",
				QueryName: "example.com",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "",
		},
		{
			name: "test DNS with type MX",
			inputDNS: DNS{
				QueryType: "MX",
				QueryName: "example.com",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    ".",
		},
		{
			name: "test DNS with type NS",
			inputDNS: DNS{
				QueryType: "NS",
				QueryName: "example.com",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "b.iana-servers.net.",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			dns := test.inputDNS
			result := &Result{}
			dns.validateAndSetDefault()
			dns.query(test.inputURL, result)
			if len(result.Errors) != 0 {
				t.Errorf("there should be no error Errors:%v", result.Errors)
			}
			if result.DNSRCode != test.expectedDNSCode {
				t.Errorf("DNSRCodePlaceHolder '%s' should have been %s", result.DNSRCode, test.expectedDNSCode)
			}

			if string(result.Body) != test.expectedBody {
				t.Errorf("got %s, expected result %s,", string(result.Body), test.expectedBody)
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

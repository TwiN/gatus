package endpoint

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/gontext"
)

func TestResolvePlaceholder(t *testing.T) {
	result := &Result{
		HTTPStatus:            200,
		IP:                    "127.0.0.1",
		Duration:              250 * time.Millisecond,
		DNSRCode:              "NOERROR",
		Connected:             true,
		CertificateExpiration: 30 * 24 * time.Hour,
		DomainExpiration:      365 * 24 * time.Hour,
		Body:                  []byte(`{"status":"success", "ts":"2m","items":[1,2,3],"user":{"name":"john","id":123}}`),
	}

	ctx := gontext.New(map[string]interface{}{
		"user_id":       "abc123",
		"session_token": "xyz789",
		"array_data":    []interface{}{"a", "b", "c"},
		"nested": map[string]interface{}{
			"value": "test",
		},
		"timestamp": "4m",
	})

	tests := []struct {
		name        string
		placeholder string
		expected    string
	}{
		// Basic placeholders
		{"status", "[STATUS]", "200"},
		{"ip", "[IP]", "127.0.0.1"},
		{"response-time", "[RESPONSE_TIME]", "250"},
		{"dns-rcode", "[DNS_RCODE]", "NOERROR"},
		{"connected", "[CONNECTED]", "true"},
		{"certificate-expiration", "[CERTIFICATE_EXPIRATION]", "2592000000"},
		{"domain-expiration", "[DOMAIN_EXPIRATION]", "31536000000"},
		{"body", "[BODY]", `{"status":"success", "ts":"2m","items":[1,2,3],"user":{"name":"john","id":123}}`},

		// Case insensitive placeholders
		{"status-lowercase", "[status]", "200"},
		{"ip-mixed-case", "[Ip]", "127.0.0.1"},

		// Function wrappers on basic placeholders
		{"len-status", "len([STATUS])", "3"},
		{"len-ip", "len([IP])", "9"},
		{"has-status", "has([STATUS])", "true"},
		{"has-empty", "has()", "false"},
		{"len-empty", "len()", "len() (INVALID)"},
		{"age-empty", "age()", "age() (INVALID)"},
		{"age-dns-rcode", "age([DNS_RCODE])", `failed to parse "NOERROR": unknown format`},

		// JSONPath expressions
		{"body-status", "[BODY].status", "success"},
		{"body-user-name", "[BODY].user.name", "john"},
		{"body-user-id", "[BODY].user.id", "123"},
		{"len-body-items", "len([BODY].items)", "3"},
		{"body-array-index", "[BODY].items[0]", "1"},
		{"has-body-status", "has([BODY].status)", "true"},
		{"has-body-missing", "has([BODY].missing)", "false"},
		{"age-body-ts", "age([BODY].ts)", "age: 120000"},
		{"age-body-status", "age([BODY].status)", `failed to parse "success": unknown format`},

		// Context placeholders
		{"context-user-id", "[CONTEXT].user_id", "abc123"},
		{"context-session-token", "[CONTEXT].session_token", "xyz789"},
		{"context-nested", "[CONTEXT].nested.value", "test"},
		{"len-context-array", "len([CONTEXT].array_data)", "3"},
		{"has-context-user-id", "has([CONTEXT].user_id)", "true"},
		{"has-context-missing", "has([CONTEXT].missing)", "false"},
		{"age-context-timestamp", "age([CONTEXT].timestamp)", "age: 240000"},

		// Invalid placeholders
		{"unknown-placeholder", "[UNKNOWN]", "[UNKNOWN]"},
		{"len-unknown", "len([UNKNOWN])", "len([UNKNOWN]) (INVALID)"},
		{"has-unknown", "has([UNKNOWN])", "false"},
		{"invalid-jsonpath", "[BODY].invalid.path", "[BODY].invalid.path (INVALID)"},

		// Literal strings
		{"literal-string", "literal", "literal"},
		{"number-string", "123", "123"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ResolvePlaceholder(test.placeholder, result, ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if strings.HasPrefix(test.expected, "age: ") {
				assertAgeInRange(t, actual, test.expected)
			} else if actual != test.expected {
				t.Errorf("expected '%s', got '%s'", test.expected, actual)
			}
		})
	}
}

func TestResolvePlaceholderWithoutContext(t *testing.T) {
	result := &Result{
		HTTPStatus: 404,
		Body:       []byte(`{"error":"not found"}`),
	}

	tests := []struct {
		name        string
		placeholder string
		expected    string
	}{
		{"status-without-context", "[STATUS]", "404"},
		{"body-without-context", "[BODY].error", "not found"},
		{"context-without-context", "[CONTEXT].user_id", "[CONTEXT].user_id"},
		{"has-context-without-context", "has([CONTEXT].user_id)", "false"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ResolvePlaceholder(test.placeholder, result, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if actual != test.expected {
				t.Errorf("expected '%s', got '%s'", test.expected, actual)
			}
		})
	}
}

func assertAgeInRange(t *testing.T, actual string, expected string) {
	expectedAge, _ := strconv.Atoi(strings.TrimPrefix(expected, "age: "))
	actualAge, err := strconv.Atoi(actual)
	if err != nil {
		t.Errorf("expected an age in milliseconds, got '%s'", actual)
	} else if actualAge < expectedAge-1000 || actualAge > expectedAge+1000 {
		t.Errorf("expected age around %d ms, got %d ms", expectedAge, actualAge)
	}
}

func TestParseAge(t *testing.T) {
	makeAge := func(duration time.Duration) string {
		return fmt.Sprintf("age: %d", duration.Milliseconds())
	}

	now := time.Now()

	absoluteDate := time.Date(2024, 10, 12, 15, 13, 6, 0, time.UTC)
	absoluteDateLocal := time.Date(2024, 10, 12, 15, 13, 6, 0, time.Local)
	absoluteDateOffset := time.Date(2024, 10, 12, 15, 13, 6, 0, time.FixedZone("Offset", 4*60*60))

	timeOnlyLocal := time.Date(now.Year(), now.Month(), now.Day(), 4, 13, 6, 0, time.Local)

	tests := []struct {
		timestamp string
		expected  string
	}{
		// RFC3339/Nano / ISO 8601 formats
		{"2024-10-12T15:13:06Z", makeAge(time.Since(absoluteDate))},
		{" 2024-10-12T15:13:06Z", makeAge(time.Since(absoluteDate))},
		{"2024-10-12T15:13:06Z ", makeAge(time.Since(absoluteDate))},
		{" 2024-10-12T15:13:06Z ", makeAge(time.Since(absoluteDate))},
		{"2024-10-12T15:13:06", makeAge(time.Since(absoluteDateLocal))},
		{"2024-10-12T15:13:06+04:00", makeAge(time.Since(absoluteDateOffset))},
		{"2024-10-12T15:13:06+0400", makeAge(time.Since(absoluteDateOffset))},
		{"2024-10-12 15:13:06Z", makeAge(time.Since(absoluteDate))},
		{"2024-10-12 15:13:06", makeAge(time.Since(absoluteDateLocal))},
		{"2024-10-12 15:13:06+04:00", makeAge(time.Since(absoluteDateOffset))},
		{"2024-10-12 15:13:06+0400", makeAge(time.Since(absoluteDateOffset))},

		// other formats
		{"Sat Oct 12 15:13:06 2024", makeAge(time.Since(absoluteDateLocal))},                     // ANSIC
		{"Sat Oct 12 15:13:06 UTC 2024", makeAge(time.Since(absoluteDate))},                      // UnixDate
		{"Sat Oct 12 15:13:06 +0400 2024", makeAge(time.Since(absoluteDateOffset))},              // RubyDate
		{"12 Oct 24 15:13 UTC", makeAge(time.Since(absoluteDate.Add(-6 * time.Second)))},         // RFC822
		{"12 Oct 24 15:13 +0400", makeAge(time.Since(absoluteDateOffset.Add(-6 * time.Second)))}, // RFC822Z
		{"Saturday, 12-Oct-24 15:13:06 UTC", makeAge(time.Since(absoluteDate))},                  // RFC850
		{"Sat, 12 Oct 2024 15:13:06.371213 UTC", makeAge(time.Since(absoluteDate))},              // RFC1123
		{"Sat, 12 Oct 2024 15:13:06.371213 +0400", makeAge(time.Since(absoluteDateOffset))},      // RFC1123Z
		{"10/12 03:13:06PM '24 +0400", makeAge(time.Since(absoluteDateOffset))},                  // Go reference time

		// common access log format
		{"12/Oct/2024 15:13:06 +0400", makeAge(time.Since(absoluteDateOffset))},
		{"12/Oct/2024:15:13:06 +0000", makeAge(time.Since(absoluteDate))},
		{"12/Oct/2024 15:13:06", makeAge(time.Since(absoluteDateLocal))},
		{"12/Oct/2024:15:13:06", makeAge(time.Since(absoluteDateLocal))},
		{"2024/10/12 15:13:06", makeAge(time.Since(absoluteDateLocal))},

		// finds timestamp in text (limited to first 128 characters)
		{" -- 2024-10-12T15:13:06Z -- ", makeAge(time.Since(absoluteDate))},
		{fmt.Sprintf("%s 2024-10-12T15:13:06Z", strings.Repeat("-", 128)), fmt.Sprintf(`failed to parse "%s": unknown format`, strings.Repeat("-", 128))},

		// custom format - currently not functional as age() call doesn't allow a parameter list.
		{"[[2006|01|02 15.04.05]], 2024|10|12 15.13.06", makeAge(time.Since(absoluteDateLocal))},
		{"[[2006|01|02 15.04.05Z0700]], 2024|10|12 15.13.06Z", makeAge(time.Since(absoluteDate))},
		{"[[2006]], 2024|10|12 15.13.06Z", `failed to parse custom layout '2006': parsing time "2024|10|12 15.13.06Z": extra text: "|10|12 15.13.06Z"`},

		// unix timestamp variants
		{fmt.Sprintf("%d", absoluteDate.Unix()), makeAge(time.Since(absoluteDate))},
		{fmt.Sprintf("%d", absoluteDate.UnixMilli()), makeAge(time.Since(absoluteDate))},
		{fmt.Sprintf("%f", float64(absoluteDate.UnixMilli())/1000.0), makeAge(time.Since(absoluteDate))},
		// unix timestamp variants with local TZ
		{fmt.Sprintf("%d", absoluteDateLocal.Unix()), makeAge(time.Since(absoluteDateLocal))},
		{fmt.Sprintf("%d", absoluteDateLocal.UnixMilli()), makeAge(time.Since(absoluteDateLocal))},
		{fmt.Sprintf("%f", float64(absoluteDateLocal.UnixMilli())/1000.0), makeAge(time.Since(absoluteDateLocal))},
		// scientific notation (when the timestamp is a JSON number, it becomes scientific notation after being stringified!)
		{"1.771548534964e+12", makeAge(time.Since(time.UnixMilli(1771548627000)) + 92036*time.Millisecond)},
		{"1.771548534964e+9", makeAge(time.Since(time.Unix(1771548627, 0)) + 92036*time.Millisecond)},

		// date only
		{"2024-10-12", makeAge(time.Since(absoluteDateLocal.Add(-(15*time.Hour + 13*time.Minute + 6*time.Second))))},

		// time only
		{"04:13:06", makeAge(time.Since(timeOnlyLocal))},
		{"4:13AM", makeAge(time.Since(timeOnlyLocal.Add(-6 * time.Second)))},

		// duration
		{"15s", makeAge(15 * time.Second)},
		{"1m15s", makeAge(75 * time.Second)},

		// future date (negative age)
		{"-24h", makeAge(-1 * 24 * time.Hour)},
		{fmt.Sprintf("%d", time.Now().Add(3*24*time.Hour).UnixMilli()), makeAge(-1 * 3 * 24 * time.Hour)},

		// error
		{"this is not a date", `failed to parse "this is not a date": unknown format`},
	}

	for _, test := range tests {
		t.Run(test.timestamp, func(t *testing.T) {
			actual := parseAge(test.timestamp)
			if strings.HasPrefix(test.expected, "age: ") {
				assertAgeInRange(t, actual, test.expected)
			} else if actual != test.expected {
				t.Errorf("expected '%s', got '%s'", test.expected, actual)
			}
		})
	}
}

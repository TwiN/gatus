package endpoint

import (
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/gontext"
)

func TestIsTemplateCondition(t *testing.T) {
	scenarios := []struct {
		condition string
		expected  bool
	}{
		{"{{eq .Status 200}}", true},
		{"  {{eq .Status 200}}  ", true},
		{"[STATUS] == 200", false},
		{"{{eq .Status 200} ", false},
		{"", false},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.condition, func(t *testing.T) {
			if actual := isTemplateCondition(scenario.condition); actual != scenario.expected {
				t.Errorf("expected %v, got %v", scenario.expected, actual)
			}
		})
	}
}

func TestCondition_evaluateTemplate(t *testing.T) {
	scenarios := []struct {
		Name            string
		Condition       Condition
		Result          *Result
		Context         *gontext.Gontext
		ExpectedSuccess bool
		ExpectedErrors  int
	}{
		{
			Name:            "status-eq-success",
			Condition:       `{{eq .Status 200}}`,
			Result:          &Result{HTTPStatus: 200},
			ExpectedSuccess: true,
		},
		{
			Name:            "status-eq-failure",
			Condition:       `{{eq .Status 200}}`,
			Result:          &Result{HTTPStatus: 500},
			ExpectedSuccess: false,
		},
		{
			Name:            "status-ne",
			Condition:       `{{ne .Status 200}}`,
			Result:          &Result{HTTPStatus: 500},
			ExpectedSuccess: true,
		},
		{
			Name:            "response-time-lt",
			Condition:       `{{lt .ResponseTime 500}}`,
			Result:          &Result{Duration: 100 * time.Millisecond},
			ExpectedSuccess: true,
		},
		{
			Name:            "response-time-ge",
			Condition:       `{{ge .ResponseTime 500}}`,
			Result:          &Result{Duration: 100 * time.Millisecond},
			ExpectedSuccess: false,
		},
		{
			// Regression test: coerceNumber used to require a duration string to parse to a
			// non-zero value before treating it as a duration, so "0ms"/"0s"/"0h" fell through to
			// string comparison and a condition like this one - meaning "any response at all" -
			// would silently never fire.
			Name:            "response-time-gt-zero-duration-string",
			Condition:       `{{gt .ResponseTime "0ms"}}`,
			Result:          &Result{Duration: 100 * time.Millisecond},
			ExpectedSuccess: true,
		},
		{
			Name:            "response-time-gt-zero-duration-string-zero-response",
			Condition:       `{{gt .ResponseTime "0ms"}}`,
			Result:          &Result{Duration: 0},
			ExpectedSuccess: false,
		},
		{
			Name:            "connected",
			Condition:       `{{eq .Connected true}}`,
			Result:          &Result{Connected: true},
			ExpectedSuccess: true,
		},
		{
			Name:            "ip",
			Condition:       `{{eq .IP "127.0.0.1"}}`,
			Result:          &Result{IP: "127.0.0.1"},
			ExpectedSuccess: true,
		},
		{
			Name:            "body-object-field",
			Condition:       `{{eq .Body.user.name "john"}}`,
			Result:          &Result{Body: []byte(`{"user":{"name":"john"}}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "body-array-index",
			Condition:       `{{eq (index .Body.data 0).id 1.0}}`,
			Result:          &Result{Body: []byte(`{"data":[{"id":1}]}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "body-non-json-raw-string",
			Condition:       `{{eq .Body "hello"}}`,
			Result:          &Result{Body: []byte(`hello`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "body-len-array",
			Condition:       `{{lt (len .Body.data) 5}}`,
			Result:          &Result{Body: []byte(`{"data":[{"id":1}]}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "body-has-true",
			Condition:       `{{has .Body "users"}}`,
			Result:          &Result{Body: []byte(`{"users":[]}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "body-has-false",
			Condition:       `{{has .Body "errors"}}`,
			Result:          &Result{Body: []byte(`{"name":"john.doe"}`)},
			ExpectedSuccess: false,
		},
		{
			// Pins the behavior of has() against a non-JSON body: buildTemplateData falls back to
			// string(result.Body) when the body isn't valid JSON, so .Body is a plain string here
			// and templateHas's default case returns false rather than doing a substring check -
			// consistent with the legacy has() semantics. If this is ever changed to do a
			// substring check instead, this test should be updated deliberately.
			Name:            "body-has-false-for-non-json-body",
			Condition:       `{{has .Body "errors"}}`,
			Result:          &Result{Body: []byte(`errors: something went wrong`)},
			ExpectedSuccess: false,
		},
		{
			Name:            "body-pat",
			Condition:       `{{pat "john*" .Body.name}}`,
			Result:          &Result{Body: []byte(`{"name":"john.doe"}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "status-any",
			Condition:       `{{any .Status 200 429}}`,
			Result:          &Result{HTTPStatus: 429},
			ExpectedSuccess: true,
		},
		{
			Name:            "status-any-false",
			Condition:       `{{any .Status 200 429}}`,
			Result:          &Result{HTTPStatus: 404},
			ExpectedSuccess: false,
		},
		{
			Name:            "headers-simple",
			Condition:       `{{eq .Headers.Location "https://example.com/"}}`,
			Result:          &Result{HTTPResponseHeaders: map[string][]string{"Location": {"https://example.com/"}}},
			ExpectedSuccess: true,
		},
		{
			Name:            "headers-hyphenated-name-via-index",
			Condition:       `{{eq (index .Headers "Content-Type") "application/json"}}`,
			Result:          &Result{HTTPResponseHeaders: map[string][]string{"Content-Type": {"application/json"}}},
			ExpectedSuccess: true,
		},
		{
			Name:            "headers-multi-value-len",
			Condition:       `{{eq (len (index .Headers "Set-Cookie")) 2}}`,
			Result:          &Result{HTTPResponseHeaders: map[string][]string{"Set-Cookie": {"a=1", "b=2"}}},
			ExpectedSuccess: true,
		},
		{
			Name:            "headers-has-true",
			Condition:       `{{has .Headers "Location"}}`,
			Result:          &Result{HTTPResponseHeaders: map[string][]string{"Location": {"https://example.com/"}}},
			ExpectedSuccess: true,
		},
		{
			Name:            "headers-has-false",
			Condition:       `{{has .Headers "Location"}}`,
			Result:          &Result{HTTPResponseHeaders: map[string][]string{}},
			ExpectedSuccess: false,
		},
		{
			Name:            "certificate-expiration-duration-string",
			Condition:       `{{gt .CertificateExpiration "48h"}}`,
			Result:          &Result{CertificateExpiration: 49 * time.Hour},
			ExpectedSuccess: true,
		},
		{
			Name:            "certificate-expiration-duration-string-failure",
			Condition:       `{{gt .CertificateExpiration "48h"}}`,
			Result:          &Result{CertificateExpiration: 1 * time.Hour},
			ExpectedSuccess: false,
		},
		{
			Name:            "domain-expiration-duration-string",
			Condition:       `{{gt .DomainExpiration "720h"}}`,
			Result:          &Result{DomainExpiration: 4000 * time.Hour},
			ExpectedSuccess: true,
		},
		{
			Name:            "context-valid",
			Condition:       `{{eq .Status .Context.expected_status}}`,
			Result:          &Result{HTTPStatus: 200},
			Context:         gontext.New(map[string]interface{}{"expected_status": 200}),
			ExpectedSuccess: true,
		},
		{
			Name:            "context-missing-key",
			Condition:       `{{has .Context "expected_status"}}`,
			Result:          &Result{HTTPStatus: 200},
			Context:         gontext.New(map[string]interface{}{"other_key": 1}),
			ExpectedSuccess: false,
		},
		{
			Name:            "non-boolean-output",
			Condition:       `{{.Status}}`,
			Result:          &Result{HTTPStatus: 200},
			ExpectedSuccess: false,
			ExpectedErrors:  1,
		},
		{
			Name:            "missing-nested-body-key-resolves-to-nil-not-error",
			Condition:       `{{eq .Body.missing.deeper 1}}`,
			Result:          &Result{Body: []byte(`{}`)},
			ExpectedSuccess: false,
		},
		{
			Name:            "execution-error-index-out-of-range",
			Condition:       `{{index .Body.data 5}}`,
			Result:          &Result{Body: []byte(`{"data":[1,2]}`)},
			ExpectedSuccess: false,
			ExpectedErrors:  1,
		},
		{
			Name:            "sprig-trim-and-upper",
			Condition:       `{{eq (upper (trim .Body.name)) "JOHN"}}`,
			Result:          &Result{Body: []byte(`{"name":"  john  "}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "sprig-contains",
			Condition:       `{{contains "doe" .Body.name}}`,
			Result:          &Result{Body: []byte(`{"name":"john.doe"}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "sprig-hasPrefix",
			Condition:       `{{hasPrefix "2." .Body.version}}`,
			Result:          &Result{Body: []byte(`{"version":"2.5.1"}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "sprig-regexMatch",
			Condition:       `{{regexMatch "^[0-9]+$" .Body.id}}`,
			Result:          &Result{Body: []byte(`{"id":"12345"}`)},
			ExpectedSuccess: true,
		},
		{
			Name:            "sprig-hasSuffix-on-header",
			Condition:       `{{hasSuffix ".json" .Headers.Location}}`,
			Result:          &Result{HTTPResponseHeaders: map[string][]string{"Location": {"https://example.com/report.json"}}},
			ExpectedSuccess: true,
		},
		{
			Name:            "sprig-splitList-with-len",
			Condition:       `{{eq (len (splitList "," .Headers.Vary)) 3}}`,
			Result:          &Result{HTTPResponseHeaders: map[string][]string{"Vary": {"Accept,Accept-Encoding,Origin"}}},
			ExpectedSuccess: true,
		},
		{
			Name:            "sprig-and-or-not-logical-combinators",
			Condition:       `{{and (eq .Status 200) (lt .ResponseTime 500)}}`,
			Result:          &Result{HTTPStatus: 200, Duration: 100 * time.Millisecond},
			ExpectedSuccess: true,
		},
		{
			// "has" is overridden by conditionFuncMap (container, key), so this is NOT
			// Sprig's list-membership has(needle, haystack) - it demonstrates that our
			// "has" takes precedence over Sprig's function of the same name.
			Name:            "custom-has-shadows-sprig-list-has",
			Condition:       `{{has (list "a" "b" "c") "b"}}`,
			Result:          &Result{},
			ExpectedSuccess: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			result := scenario.Result
			result.Errors = nil
			success := scenario.Condition.evaluate(result, false, false, scenario.Context)
			if success != scenario.ExpectedSuccess {
				t.Errorf("expected success=%v, got %v (errors: %v)", scenario.ExpectedSuccess, success, result.Errors)
			}
			if len(result.ConditionResults) != 1 {
				t.Fatalf("expected exactly one ConditionResult, got %d", len(result.ConditionResults))
			}
			// On failure (the default flags used here), the displayed condition may have
			// " (path=value, ...)" appended for the fields it referenced - see
			// TestCondition_evaluateTemplate_ResolvedFieldDisplay for the exact format. Here we
			// only assert that the raw template text itself is preserved as a prefix.
			if !strings.HasPrefix(result.ConditionResults[0].Condition, string(scenario.Condition)) {
				t.Errorf("expected displayed condition to start with the raw template text %q, got %q", scenario.Condition, result.ConditionResults[0].Condition)
			}
			if result.ConditionResults[0].Success != scenario.ExpectedSuccess {
				t.Errorf("expected ConditionResult.Success=%v, got %v", scenario.ExpectedSuccess, result.ConditionResults[0].Success)
			}
			if scenario.ExpectedErrors > 0 && len(result.Errors) != scenario.ExpectedErrors {
				t.Errorf("expected %d errors, got %d (%v)", scenario.ExpectedErrors, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestCondition_evaluateTemplate_ResolvedFieldDisplay(t *testing.T) {
	scenarios := []struct {
		Name                        string
		Condition                   Condition
		Result                      *Result
		Context                     *gontext.Gontext
		DontResolveFailedConditions bool
		ResolveSuccessfulConditions bool
		ExpectedDisplay             string
	}{
		{
			Name:            "single-field-failure-default-flags",
			Condition:       `{{eq .Status 200}}`,
			Result:          &Result{HTTPStatus: 500},
			ExpectedDisplay: `{{eq .Status 200}} (Status=500)`,
		},
		{
			Name:            "multiple-fields-failure",
			Condition:       `{{and (eq .Status 200) (lt .ResponseTime 500)}}`,
			Result:          &Result{HTTPStatus: 500, Duration: 750 * time.Millisecond},
			ExpectedDisplay: `{{and (eq .Status 200) (lt .ResponseTime 500)}} (Status=500, ResponseTime=750)`,
		},
		{
			Name:            "nested-body-field",
			Condition:       `{{eq .Body.user.name "john"}}`,
			Result:          &Result{Body: []byte(`{"user":{"name":"bob"}}`)},
			ExpectedDisplay: `{{eq .Body.user.name "john"}} (Body.user.name=bob)`,
		},
		{
			// The resolved value here is 26 chars ("https://elsewhere.example/"), one over the
			// truncation threshold, so it gets truncated too - see large-body-value-is-truncated
			// below for the dedicated truncation test.
			Name:            "header-field",
			Condition:       `{{eq .Headers.Location "https://example.com/"}}`,
			Result:          &Result{HTTPResponseHeaders: map[string][]string{"Location": {"https://elsewhere.example/"}}},
			ExpectedDisplay: `{{eq .Headers.Location "https://example.com/"}} (Headers.Location=https://elsewhere.example...(truncated))`,
		},
		{
			Name:            "context-field",
			Condition:       `{{eq .Status .Context.expected_status}}`,
			Result:          &Result{HTTPStatus: 500},
			Context:         gontext.New(map[string]interface{}{"expected_status": 200}),
			ExpectedDisplay: `{{eq .Status .Context.expected_status}} (Status=500, Context.expected_status=200)`,
		},
		{
			// .Body.foo can't be resolved for display because Body is a non-JSON raw string
			// here (traversing "foo" into a string fails); the path is silently skipped rather
			// than shown as an error, leaving the raw template text with no " (...)" suffix.
			Name:            "unresolvable-path-is-skipped",
			Condition:       `{{eq .Body.foo "1"}}`,
			Result:          &Result{Body: []byte(`not json`)},
			ExpectedDisplay: `{{eq .Body.foo "1"}}`,
		},
		{
			Name:                        "dont-resolve-failed-conditions-suppresses-display",
			Condition:                   `{{eq .Status 200}}`,
			Result:                      &Result{HTTPStatus: 500},
			DontResolveFailedConditions: true,
			ExpectedDisplay:             `{{eq .Status 200}}`,
		},
		{
			Name:            "success-with-default-flags-is-not-resolved",
			Condition:       `{{eq .Status 200}}`,
			Result:          &Result{HTTPStatus: 200},
			ExpectedDisplay: `{{eq .Status 200}}`,
		},
		{
			Name:                        "resolve-successful-conditions-shows-display-on-success",
			Condition:                   `{{eq .Status 200}}`,
			Result:                      &Result{HTTPStatus: 200},
			ResolveSuccessfulConditions: true,
			ExpectedDisplay:             `{{eq .Status 200}} (Status=200)`,
		},
		{
			// Regression test: formatFieldValue used to JSON-encode/stringify resolved values
			// verbatim with no length cap, so comparing a large non-JSON body (e.g. an HTML error
			// page) would dump the whole thing into the alert message. This mirrors the 25-char
			// truncation the legacy condition path applies (maximumLengthBeforeTruncatingWhenComparedWithPattern).
			Name:            "large-body-value-is-truncated",
			Condition:       `{{eq .Body "ok"}}`,
			Result:          &Result{Body: []byte(strings.Repeat("x", 100))},
			ExpectedDisplay: `{{eq .Body "ok"}} (Body=` + strings.Repeat("x", 25) + `...(truncated))`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Condition.evaluate(scenario.Result, scenario.DontResolveFailedConditions, scenario.ResolveSuccessfulConditions, scenario.Context)
			if len(scenario.Result.ConditionResults) != 1 {
				t.Fatalf("expected exactly one ConditionResult, got %d", len(scenario.Result.ConditionResults))
			}
			if actual := scenario.Result.ConditionResults[0].Condition; actual != scenario.ExpectedDisplay {
				t.Errorf("expected display %q, got %q", scenario.ExpectedDisplay, actual)
			}
		})
	}
}

func TestCondition_ValidateTemplate(t *testing.T) {
	scenarios := []struct {
		Name        string
		Condition   Condition
		ExpectError bool
	}{
		{Name: "valid-simple", Condition: `{{eq .Status 200}}`, ExpectError: false},
		{Name: "valid-body-index-against-empty-body", Condition: `{{eq (index .Body.data 0).id 1}}`, ExpectError: false},
		{Name: "unclosed-action", Condition: `{{eq .Status 200`, ExpectError: true},
		{Name: "unknown-function", Condition: `{{frobnicate .Status 200}}`, ExpectError: true},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Condition.Validate()
			if scenario.ExpectError && err == nil {
				t.Error("expected an error, got nil")
			}
			if !scenario.ExpectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestCondition_hasPlaceholder_templateStyle(t *testing.T) {
	scenarios := []struct {
		Name                       string
		Condition                  Condition
		ExpectedHasBody            bool
		ExpectedHasHeaders         bool
		ExpectedHasDomainExpirtion bool
		ExpectedHasIP              bool
	}{
		{
			Name:            "body-only",
			Condition:       `{{eq .Body.name "john"}}`,
			ExpectedHasBody: true,
		},
		{
			Name:               "headers-only",
			Condition:          `{{has .Headers "Location"}}`,
			ExpectedHasHeaders: true,
		},
		{
			Name:                       "domain-expiration-only",
			Condition:                  `{{gt .DomainExpiration "720h"}}`,
			ExpectedHasDomainExpirtion: true,
		},
		{
			Name:          "ip-only",
			Condition:     `{{eq .IP "127.0.0.1"}}`,
			ExpectedHasIP: true,
		},
		{
			Name:      "status-only-no-side-effects",
			Condition: `{{eq .Status 200}}`,
		},
		{
			Name:               "body-and-headers-combined",
			Condition:          `{{and (eq .Body.name "john") (has .Headers "Location")}}`,
			ExpectedHasBody:    true,
			ExpectedHasHeaders: true,
		},
		{
			// A variable aliasing a field directly is tracked correctly, because the ".Body" on
			// the right-hand side of the declaration is itself a FieldNode.
			Name:            "variable-aliasing-a-field-directly-is-tracked",
			Condition:       `{{$b := .Body}}{{if $b}}true{{else}}false{{end}}`,
			ExpectedHasBody: true,
		},
		{
			// Regression test: a variable that aliases the whole "." and is then dotted into
			// (e.g. "$root.Body") can't be resolved back to a field path without full data-flow
			// analysis, since VariableNode.Ident is ["$root", "Body"] rather than a FieldNode.
			// collectFieldPaths detects this shape and conservatively marks every gated field as
			// referenced, so the corresponding reads (body, headers, domain expiration, IP) aren't
			// skipped.
			Name:                       "variable-dotted-into-conservatively-flags-all-gated-fields",
			Condition:                  `{{$root := .}}{{if $root.Body}}true{{else}}false{{end}}`,
			ExpectedHasBody:            true,
			ExpectedHasHeaders:         true,
			ExpectedHasDomainExpirtion: true,
			ExpectedHasIP:              true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if got := scenario.Condition.hasBodyPlaceholder(); got != scenario.ExpectedHasBody {
				t.Errorf("hasBodyPlaceholder: expected %v, got %v", scenario.ExpectedHasBody, got)
			}
			if got := scenario.Condition.hasHeadersPlaceholder(); got != scenario.ExpectedHasHeaders {
				t.Errorf("hasHeadersPlaceholder: expected %v, got %v", scenario.ExpectedHasHeaders, got)
			}
			if got := scenario.Condition.hasDomainExpirationPlaceholder(); got != scenario.ExpectedHasDomainExpirtion {
				t.Errorf("hasDomainExpirationPlaceholder: expected %v, got %v", scenario.ExpectedHasDomainExpirtion, got)
			}
			if got := scenario.Condition.hasIPPlaceholder(); got != scenario.ExpectedHasIP {
				t.Errorf("hasIPPlaceholder: expected %v, got %v", scenario.ExpectedHasIP, got)
			}
		})
	}
}

func TestEndpoint_needsToReadBodyAndHeaders_templateStyle(t *testing.T) {
	e := &Endpoint{
		Conditions: []Condition{`{{eq .Body.name "john"}}`, `{{has .Headers "Location"}}`},
	}
	if !e.needsToReadBody() {
		t.Error("expected needsToReadBody to be true because a condition references .Body")
	}
	if !e.needsToReadHeaders() {
		t.Error("expected needsToReadHeaders to be true because a condition references .Headers")
	}
	e2 := &Endpoint{
		Conditions: []Condition{`{{eq .Status 200}}`},
	}
	if e2.needsToReadBody() {
		t.Error("expected needsToReadBody to be false, no condition references .Body")
	}
	if e2.needsToReadHeaders() {
		t.Error("expected needsToReadHeaders to be false, no condition references .Headers")
	}
	e3 := &Endpoint{
		Conditions: []Condition{`{{gt .DomainExpiration "720h"}}`},
	}
	if !e3.needsToRetrieveDomainExpiration() {
		t.Error("expected needsToRetrieveDomainExpiration to be true because a condition references .DomainExpiration")
	}
	e4 := &Endpoint{
		Conditions: []Condition{`{{eq .IP "127.0.0.1"}}`},
	}
	if !e4.needsToRetrieveIP() {
		t.Error("expected needsToRetrieveIP to be true because a condition references .IP")
	}
}

func TestEndpoint_ValidateAndSetDefaults_mixedLegacyAndTemplateConditions(t *testing.T) {
	e := &Endpoint{
		Name:       "mixed",
		URL:        "https://example.com",
		Conditions: []Condition{`[STATUS] == 200`, `{{lt .ResponseTime 500}}`},
	}
	if err := e.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("expected no error mixing legacy and template conditions, got %v", err)
	}
}

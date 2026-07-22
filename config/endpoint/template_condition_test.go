package endpoint

import (
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
			if result.ConditionResults[0].Condition != string(scenario.Condition) {
				t.Errorf("expected displayed condition to be the raw template text %q, got %q", scenario.Condition, result.ConditionResults[0].Condition)
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

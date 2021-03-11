package core

import (
	"strconv"
	"testing"
	"time"
)

func TestCondition_evaluate(t *testing.T) {
	type scenario struct {
		Name            string
		Condition       Condition
		Result          *Result
		ExpectedSuccess bool
		ExpectedOutput  string
	}
	scenarios := []scenario{
		{
			Name:            "ip",
			Condition:       Condition("[IP] == 127.0.0.1"),
			Result:          &Result{IP: "127.0.0.1"},
			ExpectedSuccess: true,
			ExpectedOutput:  "[IP] == 127.0.0.1",
		},
		{
			Name:            "status",
			Condition:       Condition("[STATUS] == 200"),
			Result:          &Result{HTTPStatus: 200},
			ExpectedSuccess: true,
			ExpectedOutput:  "[STATUS] == 200",
		},
		{
			Name:            "status-failure",
			Condition:       Condition("[STATUS] == 200"),
			Result:          &Result{HTTPStatus: 500},
			ExpectedSuccess: false,
			ExpectedOutput:  "[STATUS] (500) == 200",
		},
		{
			Name:            "status-using-less-than",
			Condition:       Condition("[STATUS] < 300"),
			Result:          &Result{HTTPStatus: 201},
			ExpectedSuccess: true,
			ExpectedOutput:  "[STATUS] < 300",
		},
		{
			Name:            "status-using-less-than-failure",
			Condition:       Condition("[STATUS] < 300"),
			Result:          &Result{HTTPStatus: 404},
			ExpectedSuccess: false,
			ExpectedOutput:  "[STATUS] (404) < 300",
		},
		{
			Name:            "response-time-using-less-than",
			Condition:       Condition("[RESPONSE_TIME] < 500"),
			Result:          &Result{Duration: 50 * time.Millisecond},
			ExpectedSuccess: true,
			ExpectedOutput:  "[RESPONSE_TIME] < 500",
		},
		{
			Name:            "response-time-using-less-than-with-duration",
			Condition:       Condition("[RESPONSE_TIME] < 1s"),
			Result:          &Result{Duration: 50 * time.Millisecond},
			ExpectedSuccess: true,
			ExpectedOutput:  "[RESPONSE_TIME] < 1s",
		},
		{
			Name:            "response-time-using-less-than-invalid",
			Condition:       Condition("[RESPONSE_TIME] < potato"),
			Result:          &Result{Duration: 50 * time.Millisecond},
			ExpectedSuccess: false,
			ExpectedOutput:  "[RESPONSE_TIME] (50) < potato (0)", // Non-numerical values automatically resolve to 0
		},
		{
			Name:            "response-time-using-greater-than",
			Condition:       Condition("[RESPONSE_TIME] > 500"),
			Result:          &Result{Duration: 750 * time.Millisecond},
			ExpectedSuccess: true,
			ExpectedOutput:  "[RESPONSE_TIME] > 500",
		},
		{
			Name:            "response-time-using-greater-than-with-duration",
			Condition:       Condition("[RESPONSE_TIME] > 1s"),
			Result:          &Result{Duration: 2 * time.Second},
			ExpectedSuccess: true,
			ExpectedOutput:  "[RESPONSE_TIME] > 1s",
		},
		{
			Name:            "response-time-using-greater-than-or-equal-to-equal",
			Condition:       Condition("[RESPONSE_TIME] >= 500"),
			Result:          &Result{Duration: 500 * time.Millisecond},
			ExpectedSuccess: true,
			ExpectedOutput:  "[RESPONSE_TIME] >= 500",
		},
		{
			Name:            "response-time-using-greater-than-or-equal-to-greater",
			Condition:       Condition("[RESPONSE_TIME] >= 500"),
			Result:          &Result{Duration: 499 * time.Millisecond},
			ExpectedSuccess: false,
			ExpectedOutput:  "[RESPONSE_TIME] (499) >= 500",
		},
		{
			Name:            "response-time-using-greater-than-or-equal-to-failure",
			Condition:       Condition("[RESPONSE_TIME] >= 500"),
			Result:          &Result{Duration: 499 * time.Millisecond},
			ExpectedSuccess: false,
			ExpectedOutput:  "[RESPONSE_TIME] (499) >= 500",
		},
		{
			Name:            "response-time-using-less-than-or-equal-to-equal",
			Condition:       Condition("[RESPONSE_TIME] <= 500"),
			Result:          &Result{Duration: 500 * time.Millisecond},
			ExpectedSuccess: true,
			ExpectedOutput:  "[RESPONSE_TIME] <= 500",
		},
		{
			Name:            "response-time-using-less-than-or-equal-to-less",
			Condition:       Condition("[RESPONSE_TIME] <= 500"),
			Result:          &Result{Duration: 25 * time.Millisecond},
			ExpectedSuccess: true,
			ExpectedOutput:  "[RESPONSE_TIME] <= 500",
		},
		{
			Name:            "response-time-using-less-than-or-equal-to-failure",
			Condition:       Condition("[RESPONSE_TIME] <= 500"),
			Result:          &Result{Duration: 750 * time.Millisecond},
			ExpectedSuccess: false,
			ExpectedOutput:  "[RESPONSE_TIME] (750) <= 500",
		},
		{
			Name:            "body",
			Condition:       Condition("[BODY] == test"),
			Result:          &Result{body: []byte("test")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] == test",
		},
		{
			Name:            "body-jsonpath",
			Condition:       Condition("[BODY].status == UP"),
			Result:          &Result{body: []byte("{\"status\":\"UP\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].status == UP",
		},
		{
			Name:            "body-jsonpath-complex",
			Condition:       Condition("[BODY].data.name == john"),
			Result:          &Result{body: []byte("{\"data\": {\"id\": 1, \"name\": \"john\"}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data.name == john",
		},
		{
			Name:            "body-jsonpath-complex-invalid",
			Condition:       Condition("[BODY].data.name == john"),
			Result:          &Result{body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].data.name (INVALID) == john",
		},
		{
			Name:            "body-jsonpath-complex-len",
			Condition:       Condition("len([BODY].data.name) == 4"),
			Result:          &Result{body: []byte("{\"data\": {\"name\": \"john\"}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].data.name) == 4",
		},
		{
			Name:            "body-jsonpath-complex-len-invalid",
			Condition:       Condition("len([BODY].data.name) == john"),
			Result:          &Result{body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "len([BODY].data.name) (INVALID) == john",
		},
		{
			Name:            "body-jsonpath-double-placeholder",
			Condition:       Condition("[BODY].user.firstName != [BODY].user.lastName"),
			Result:          &Result{body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].user.firstName != [BODY].user.lastName",
		},
		{
			Name:            "body-jsonpath-double-placeholder-failure",
			Condition:       Condition("[BODY].user.firstName == [BODY].user.lastName"),
			Result:          &Result{body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].user.firstName (john) == [BODY].user.lastName (doe)",
		},
		{
			Name:            "body-jsonpath-complex-int",
			Condition:       Condition("[BODY].data.id == 1"),
			Result:          &Result{body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data.id == 1",
		},
		{
			Name:            "body-jsonpath-complex-array-int",
			Condition:       Condition("[BODY].data[1].id == 2"),
			Result:          &Result{body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data[1].id == 2",
		},
		{
			Name:            "body-jsonpath-complex-int-using-greater-than",
			Condition:       Condition("[BODY].data.id > 0"),
			Result:          &Result{body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data.id > 0",
		},
		{
			Name:            "body-jsonpath-complex-int-using-greater-than-failure",
			Condition:       Condition("[BODY].data.id > 5"),
			Result:          &Result{body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].data.id (1) > 5",
		},
		{
			Name:            "body-jsonpath-complex-int-using-less-than",
			Condition:       Condition("[BODY].data.id < 5"),
			Result:          &Result{body: []byte("{\"data\": {\"id\": 2}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data.id < 5",
		},
		{
			Name:            "body-jsonpath-complex-int-using-less-than-failure",
			Condition:       Condition("[BODY].data.id < 5"),
			Result:          &Result{body: []byte("{\"data\": {\"id\": 10}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].data.id (10) < 5",
		},
		{
			Name:            "body-len-array",
			Condition:       Condition("len([BODY].data) == 3"),
			Result:          &Result{body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].data) == 3",
		},
		{
			Name:            "body-len-array-invalid",
			Condition:       Condition("len([BODY].data) == 8"),
			Result:          &Result{body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "len([BODY].data) (INVALID) == 8",
		},
		{
			Name:            "body-len-string",
			Condition:       Condition("len([BODY].name) == 8"),
			Result:          &Result{body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].name) == 8",
		},
		{
			Name:            "body-pattern",
			Condition:       Condition("[BODY] == pat(*john*)"),
			Result:          &Result{body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] == pat(*john*)",
		},
		{
			Name:            "body-pattern-2",
			Condition:       Condition("[BODY].name == pat(john*)"),
			Result:          &Result{body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].name == pat(john*)",
		},
		{
			Name:            "body-pattern-failure",
			Condition:       Condition("[BODY].name == pat(bob*)"),
			Result:          &Result{body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].name (john.doe) == pat(bob*)",
		},
		{
			Name:            "body-pattern-html",
			Condition:       Condition("[BODY] == pat(*<div id=\"user\">john.doe</div>*)"),
			Result:          &Result{body: []byte(`<!DOCTYPE html><html lang="en"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head><body><div id="user">john.doe</div></body></html>`)},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] == pat(*<div id=\"user\">john.doe</div>*)",
		},
		{
			Name:            "ip-pattern",
			Condition:       Condition("[IP] == pat(10.*)"),
			Result:          &Result{IP: "10.0.0.0"},
			ExpectedSuccess: true,
			ExpectedOutput:  "[IP] == pat(10.*)",
		},
		{
			Name:            "ip-pattern-failure",
			Condition:       Condition("[IP] == pat(10.*)"),
			Result:          &Result{IP: "255.255.255.255"},
			ExpectedSuccess: false,
			ExpectedOutput:  "[IP] (255.255.255.255) == pat(10.*)",
		},
		{
			Name:            "status-pattern",
			Condition:       Condition("[STATUS] == pat(4*)"),
			Result:          &Result{HTTPStatus: 404},
			ExpectedSuccess: true,
			ExpectedOutput:  "[STATUS] == pat(4*)",
		},
		{
			Name:            "status-pattern-failure",
			Condition:       Condition("[STATUS] == pat(4*)"),
			Result:          &Result{HTTPStatus: 200},
			ExpectedSuccess: false,
			ExpectedOutput:  "[STATUS] (200) == pat(4*)",
		},
		{
			Name:            "body-any",
			Condition:       Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:          &Result{body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].name == any(john.doe, jane.doe)",
		},
		{
			Name:            "body-any-2",
			Condition:       Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:          &Result{body: []byte("{\"name\": \"jane.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].name == any(john.doe, jane.doe)",
		},
		{
			Name:            "body-any-failure",
			Condition:       Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:          &Result{body: []byte("{\"name\": \"bob\"}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].name (bob) == any(john.doe, jane.doe)",
		},
		{
			Name:            "status-any",
			Condition:       Condition("[STATUS] == any(200, 429)"),
			Result:          &Result{HTTPStatus: 200},
			ExpectedSuccess: true,
			ExpectedOutput:  "[STATUS] == any(200, 429)",
		},
		{
			Name:            "status-any-2",
			Condition:       Condition("[STATUS] == any(200, 429)"),
			Result:          &Result{HTTPStatus: 429},
			ExpectedSuccess: true,
			ExpectedOutput:  "[STATUS] == any(200, 429)",
		},
		{
			Name:            "status-any-reverse",
			Condition:       Condition("any(200, 429) == [STATUS]"),
			Result:          &Result{HTTPStatus: 429},
			ExpectedSuccess: true,
			ExpectedOutput:  "any(200, 429) == [STATUS]",
		},
		{
			Name:            "status-any-failure",
			Condition:       Condition("[STATUS] == any(200, 429)"),
			Result:          &Result{HTTPStatus: 404},
			ExpectedSuccess: false,
			ExpectedOutput:  "[STATUS] (404) == any(200, 429)",
		},
		{
			Name:            "connected",
			Condition:       Condition("[CONNECTED] == true"),
			Result:          &Result{Connected: true},
			ExpectedSuccess: true,
			ExpectedOutput:  "[CONNECTED] == true",
		},
		{
			Name:            "connected-failure",
			Condition:       Condition("[CONNECTED] == true"),
			Result:          &Result{Connected: false},
			ExpectedSuccess: false,
			ExpectedOutput:  "[CONNECTED] (false) == true",
		},
		{
			Name:            "certificate-expiration-not-set",
			Condition:       Condition("[CERTIFICATE_EXPIRATION] == 0"),
			Result:          &Result{},
			ExpectedSuccess: true,
			ExpectedOutput:  "[CERTIFICATE_EXPIRATION] == 0",
		},
		{
			Name:            "certificate-expiration-greater-than-numerical",
			Condition:       Condition("[CERTIFICATE_EXPIRATION] > " + strconv.FormatInt((time.Hour*24*28).Milliseconds(), 10)),
			Result:          &Result{CertificateExpiration: time.Hour * 24 * 60},
			ExpectedSuccess: true,
			ExpectedOutput:  "[CERTIFICATE_EXPIRATION] > 2419200000",
		},
		{
			Name:            "certificate-expiration-greater-than-numerical-failure",
			Condition:       Condition("[CERTIFICATE_EXPIRATION] > " + strconv.FormatInt((time.Hour*24*28).Milliseconds(), 10)),
			Result:          &Result{CertificateExpiration: time.Hour * 24 * 14},
			ExpectedSuccess: false,
			ExpectedOutput:  "[CERTIFICATE_EXPIRATION] (1209600000) > 2419200000",
		},
		{
			Name:            "certificate-expiration-greater-than-duration",
			Condition:       Condition("[CERTIFICATE_EXPIRATION] > 12h"),
			Result:          &Result{CertificateExpiration: 24 * time.Hour},
			ExpectedSuccess: true,
			ExpectedOutput:  "[CERTIFICATE_EXPIRATION] > 12h",
		},
		{
			Name:            "certificate-expiration-greater-than-duration",
			Condition:       Condition("[CERTIFICATE_EXPIRATION] > 48h"),
			Result:          &Result{CertificateExpiration: 24 * time.Hour},
			ExpectedSuccess: false,
			ExpectedOutput:  "[CERTIFICATE_EXPIRATION] (86400000) > 48h (172800000)",
		},
		{
			Name:            "has",
			Condition:       Condition("has([BODY].errors) == false"),
			Result:          &Result{body: []byte("{}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "has([BODY].errors) == false",
		},
		{
			Name:            "has-failure",
			Condition:       Condition("has([BODY].errors) == false"),
			Result:          &Result{body: []byte("{\"errors\": [\"1\"]}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "has([BODY].errors) (true) == false",
		},
		{
			Name:            "no-placeholders",
			Condition:       Condition("1 == 2"),
			Result:          &Result{},
			ExpectedSuccess: false,
			ExpectedOutput:  "1 == 2",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Condition.evaluate(scenario.Result)
			if scenario.Result.ConditionResults[0].Success != scenario.ExpectedSuccess {
				t.Errorf("Condition '%s' should have been success=%v", scenario.Condition, scenario.ExpectedSuccess)
			}
			if scenario.Result.ConditionResults[0].Condition != scenario.ExpectedOutput {
				t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", scenario.Condition, scenario.ExpectedOutput, scenario.Result.ConditionResults[0].Condition)
			}
		})
	}
}

func TestCondition_evaluateWithInvalidOperator(t *testing.T) {
	condition := Condition("[STATUS] ? 201")
	result := &Result{HTTPStatus: 201}
	condition.evaluate(result)
	if result.Success {
		t.Error("condition was invalid, result should've been a failure")
	}
	if len(result.Errors) != 1 {
		t.Error("condition was invalid, result should've had an error")
	}
}

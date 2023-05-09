package core

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestCondition_Validate(t *testing.T) {
	scenarios := []struct {
		condition   Condition
		expectedErr error
	}{
		{condition: "[STATUS] == 200", expectedErr: nil},
		{condition: "[STATUS] != 200", expectedErr: nil},
		{condition: "[STATUS] <= 200", expectedErr: nil},
		{condition: "[STATUS] >= 200", expectedErr: nil},
		{condition: "[STATUS] < 200", expectedErr: nil},
		{condition: "[STATUS] > 200", expectedErr: nil},
		{condition: "[STATUS] == any(200, 201, 202, 203)", expectedErr: nil},
		{condition: "[STATUS] == [BODY].status", expectedErr: nil},
		{condition: "[CONNECTED] == true", expectedErr: nil},
		{condition: "[RESPONSE_TIME] < 500", expectedErr: nil},
		{condition: "[IP] == 127.0.0.1", expectedErr: nil},
		{condition: "[BODY] == 1", expectedErr: nil},
		{condition: "[BODY].test == wat", expectedErr: nil},
		{condition: "[BODY].test.wat == wat", expectedErr: nil},
		{condition: "[BODY].age == [BODY].id", expectedErr: nil},
		{condition: "[BODY].users[0].id == 1", expectedErr: nil},
		{condition: "len([BODY].users) == 100", expectedErr: nil},
		{condition: "len([BODY].data) < 5", expectedErr: nil},
		{condition: "has([BODY].errors) == false", expectedErr: nil},
		{condition: "has([BODY].users[0].name) == true", expectedErr: nil},
		{condition: "[BODY].name == pat(john*)", expectedErr: nil},
		{condition: "[CERTIFICATE_EXPIRATION] > 48h", expectedErr: nil},
		{condition: "[DOMAIN_EXPIRATION] > 720h", expectedErr: nil},
		{condition: "raw == raw", expectedErr: nil},
		{condition: "[STATUS] ? 201", expectedErr: errors.New("invalid condition: [STATUS] ? 201")},
		{condition: "[STATUS]==201", expectedErr: errors.New("invalid condition: [STATUS]==201")},
		{condition: "[STATUS] = = 201", expectedErr: errors.New("invalid condition: [STATUS] = = 201")},
		{condition: "[STATUS] ==", expectedErr: errors.New("invalid condition: [STATUS] ==")},
		{condition: "[STATUS]", expectedErr: errors.New("invalid condition: [STATUS]")},
		// FIXME: Should return an error, but doesn't because jsonpath isn't evaluated due to body being empty in Condition.Validate()
		//{condition: "len([BODY].users == 100", expectedErr: nil},
	}
	for _, scenario := range scenarios {
		t.Run(string(scenario.condition), func(t *testing.T) {
			if err := scenario.condition.Validate(); fmt.Sprint(err) != fmt.Sprint(scenario.expectedErr) {
				t.Errorf("expected err %v, got %v", scenario.expectedErr, err)
			}
		})
	}
}

func TestCondition_evaluate(t *testing.T) {
	scenarios := []struct {
		Name                        string
		Condition                   Condition
		Result                      *Result
		DontResolveFailedConditions bool
		ExpectedSuccess             bool
		ExpectedOutput              string
	}{
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
			Result:          &Result{Body: []byte("test")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] == test",
		},
		{
			Name:            "body-numerical-equal",
			Condition:       Condition("[BODY] == 123"),
			Result:          &Result{Body: []byte("123")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] == 123",
		},
		{
			Name:            "body-numerical-less-than",
			Condition:       Condition("[BODY] < 124"),
			Result:          &Result{Body: []byte("123")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] < 124",
		},
		{
			Name:            "body-numerical-greater-than",
			Condition:       Condition("[BODY] > 122"),
			Result:          &Result{Body: []byte("123")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] > 122",
		},
		{
			Name:            "body-numerical-greater-than-failure",
			Condition:       Condition("[BODY] > 123"),
			Result:          &Result{Body: []byte("100")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY] (100) > 123",
		},
		{
			Name:            "body-jsonpath",
			Condition:       Condition("[BODY].status == UP"),
			Result:          &Result{Body: []byte("{\"status\":\"UP\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].status == UP",
		},
		{
			Name:            "body-jsonpath-complex",
			Condition:       Condition("[BODY].data.name == john"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 1, \"name\": \"john\"}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data.name == john",
		},
		{
			Name:            "body-jsonpath-complex-invalid",
			Condition:       Condition("[BODY].data.name == john"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].data.name (INVALID) == john",
		},
		{
			Name:            "body-jsonpath-complex-len-invalid",
			Condition:       Condition("len([BODY].data.name) == john"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "len([BODY].data.name) (INVALID) == john",
		},
		{
			Name:            "body-jsonpath-double-placeholder",
			Condition:       Condition("[BODY].user.firstName != [BODY].user.lastName"),
			Result:          &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].user.firstName != [BODY].user.lastName",
		},
		{
			Name:            "body-jsonpath-double-placeholder-failure",
			Condition:       Condition("[BODY].user.firstName == [BODY].user.lastName"),
			Result:          &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].user.firstName (john) == [BODY].user.lastName (doe)",
		},
		{
			Name:            "body-jsonpath-when-body-is-array",
			Condition:       Condition("[BODY][0].id == 1"),
			Result:          &Result{Body: []byte("[{\"id\": 1}, {\"id\": 2}]")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY][0].id == 1",
		},
		{
			Name:            "body-jsonpath-when-body-is-array-but-actual-body-is-not",
			Condition:       Condition("[BODY][0].name == test"),
			Result:          &Result{Body: []byte("{\"statusCode\": 500, \"message\": \"Internal Server Error\"}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY][0].name (INVALID) == test",
		},
		{
			Name:            "body-jsonpath-complex-int",
			Condition:       Condition("[BODY].data.id == 1"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data.id == 1",
		},
		{
			Name:            "body-jsonpath-complex-array-int",
			Condition:       Condition("[BODY].data[1].id == 2"),
			Result:          &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data[1].id == 2",
		},
		{
			Name:            "body-jsonpath-complex-int-using-greater-than",
			Condition:       Condition("[BODY].data.id > 0"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data.id > 0",
		},
		{
			Name:            "body-jsonpath-complex-int-using-greater-than-failure",
			Condition:       Condition("[BODY].data.id > 5"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].data.id (1) > 5",
		},
		{
			Name:            "body-jsonpath-float-using-greater-than-issue433", // As of v5.3.1, Gatus will convert a float to an int. We're losing precision, but it's better than just returning 0
			Condition:       Condition("[BODY].balance > 100"),
			Result:          &Result{Body: []byte(`{"balance": "123.40000000000005"}`)},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].balance > 100",
		},
		{
			Name:            "body-jsonpath-complex-int-using-less-than",
			Condition:       Condition("[BODY].data.id < 5"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 2}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data.id < 5",
		},
		{
			Name:            "body-jsonpath-complex-int-using-less-than-failure",
			Condition:       Condition("[BODY].data.id < 5"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 10}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].data.id (10) < 5",
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
			Name:            "no-placeholders",
			Condition:       Condition("1 == 2"),
			Result:          &Result{},
			ExpectedSuccess: false,
			ExpectedOutput:  "1 == 2",
		},
        {
            Name:            "exit-code-zero",
            Condition:       Condition("[EXIT_CODE] == 0"),
            Result:          &Result{ExitCode: 0},
            ExpectedSuccess: true,
            ExpectedOutput:  "[EXIT_CODE] == 0",
        },
        {
            Name:            "exit-code-zero-failure",
            Condition:       Condition("[EXIT_CODE] == 0"),
            Result:          &Result{ExitCode: 1},
            ExpectedSuccess: false,
            ExpectedOutput:  "[EXIT_CODE] (1) == 0",
        },
        {
            Name:            "exit-code-greater-than-zero",
            Condition:       Condition("[EXIT_CODE] > 0"),
            Result:          &Result{ExitCode: 1},
            ExpectedSuccess: true,
            ExpectedOutput:  "[EXIT_CODE] > 0",
        },
        {
            Name:            "exit-code-greater-than-zero-failure",
            Condition:       Condition("[EXIT_CODE] > 0"),
            Result:          &Result{ExitCode: 0},
            ExpectedSuccess: false,
            ExpectedOutput:  "[EXIT_CODE] (0) > 0",
        },
        {
            Name:            "exit-code-less-than-zero",
            Condition:       Condition("[EXIT_CODE] < 0"),
            Result:          &Result{ExitCode: -1},
            ExpectedSuccess: true,
            ExpectedOutput:  "[EXIT_CODE] < 0",
        },
        {
            Name:            "exit-code-less-than-zero-failure",
            Condition:       Condition("[EXIT_CODE] < 0"),
            Result:          &Result{ExitCode: 0},
            ExpectedSuccess: false,
            ExpectedOutput:  "[EXIT_CODE] (0) < 0",
        },
        {
            Name:            "exit-code-not-zero",
            Condition:       Condition("[EXIT_CODE] != 0"),
            Result:          &Result{ExitCode: 1},
            ExpectedSuccess: true,
            ExpectedOutput:  "[EXIT_CODE] != 0",
        },
        {
            Name:            "exit-code-not-zero-failure",
            Condition:       Condition("[EXIT_CODE] != 0"),
            Result:          &Result{ExitCode: 0},
            ExpectedSuccess: false,
            ExpectedOutput:  "[EXIT_CODE] (0) != 0",
        },
		///////////////
		// Functions //
		///////////////
		// len
		{
			Name:            "len-body-jsonpath-complex",
			Condition:       Condition("len([BODY].data.name) == 4"),
			Result:          &Result{Body: []byte("{\"data\": {\"name\": \"john\"}}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].data.name) == 4",
		},
		{
			Name:            "len-body-array",
			Condition:       Condition("len([BODY]) == 3"),
			Result:          &Result{Body: []byte("[{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY]) == 3",
		},
		{
			Name:            "len-body-keyed-array",
			Condition:       Condition("len([BODY].data) == 3"),
			Result:          &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].data) == 3",
		},
		{
			Name:            "len-body-array-invalid",
			Condition:       Condition("len([BODY].data) == 8"),
			Result:          &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "len([BODY].data) (INVALID) == 8",
		},
		{
			Name:            "len-body-string",
			Condition:       Condition("len([BODY]) == 8"),
			Result:          &Result{Body: []byte("john.doe")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY]) == 8",
		},
		{
			Name:            "len-body-keyed-string",
			Condition:       Condition("len([BODY].name) == 8"),
			Result:          &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].name) == 8",
		},
		{
			Name:            "len-body-keyed-int",
			Condition:       Condition("len([BODY].age) == 2"),
			Result:          &Result{Body: []byte(`{"age":18}`)},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].age) == 2",
		},
		{
			Name:            "len-body-keyed-bool",
			Condition:       Condition("len([BODY].adult) == 4"),
			Result:          &Result{Body: []byte(`{"adult":true}`)},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].adult) == 4",
		},
		{
			Name:            "len-body-object-inside-array",
			Condition:       Condition("len([BODY][0]) == 23"),
			Result:          &Result{Body: []byte(`[{"age":18,"adult":true}]`)},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY][0]) == 23",
		},
		{
			Name:            "len-body-object-keyed-int-inside-array",
			Condition:       Condition("len([BODY][0].age) == 2"),
			Result:          &Result{Body: []byte(`[{"age":18,"adult":true}]`)},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY][0].age) == 2",
		},
		{
			Name:            "len-body-keyed-bool-inside-array",
			Condition:       Condition("len([BODY][0].adult) == 4"),
			Result:          &Result{Body: []byte(`[{"age":18,"adult":true}]`)},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY][0].adult) == 4",
		},
		{
			Name:            "len-body-object",
			Condition:       Condition("len([BODY]) == 20"),
			Result:          &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY]) == 20",
		},
		// pat
		{
			Name:            "pat-body-1",
			Condition:       Condition("[BODY] == pat(*john*)"),
			Result:          &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] == pat(*john*)",
		},
		{
			Name:            "pat-body-2",
			Condition:       Condition("[BODY].name == pat(john*)"),
			Result:          &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].name == pat(john*)",
		},
		{
			Name:            "pat-body-failure",
			Condition:       Condition("[BODY].name == pat(bob*)"),
			Result:          &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].name (john.doe) == pat(bob*)",
		},
		{
			Name:            "pat-body-html",
			Condition:       Condition("[BODY] == pat(*<div id=\"user\">john.doe</div>*)"),
			Result:          &Result{Body: []byte(`<!DOCTYPE html><html lang="en"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head><body><div id="user">john.doe</div></body></html>`)},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY] == pat(*<div id=\"user\">john.doe</div>*)",
		},
		{
			Name:            "pat-body-html-failure",
			Condition:       Condition("[BODY] == pat(*<div id=\"user\">john.doe</div>*)"),
			Result:          &Result{Body: []byte(`<!DOCTYPE html><html lang="en"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head><body><div id="user">jane.doe</div></body></html>`)},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY] (<!DOCTYPE html><html lang...(truncated)) == pat(*<div id=\"user\">john.doe</div>*)",
		},
		{
			Name:            "pat-body-html-failure-alt",
			Condition:       Condition("pat(*<div id=\"user\">john.doe</div>*) == [BODY]"),
			Result:          &Result{Body: []byte(`<!DOCTYPE html><html lang="en"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head><body><div id="user">jane.doe</div></body></html>`)},
			ExpectedSuccess: false,
			ExpectedOutput:  "pat(*<div id=\"user\">john.doe</div>*) == [BODY] (<!DOCTYPE html><html lang...(truncated))",
		},
		{
			Name:            "pat-body-in-array",
			Condition:       Condition("[BODY].data == pat(*Whatever*)"),
			Result:          &Result{Body: []byte("{\"data\": [\"hello\", \"world\", \"Whatever\"]}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == pat(*Whatever*)",
		},
		{
			Name:            "pat-ip",
			Condition:       Condition("[IP] == pat(10.*)"),
			Result:          &Result{IP: "10.0.0.0"},
			ExpectedSuccess: true,
			ExpectedOutput:  "[IP] == pat(10.*)",
		},
		{
			Name:            "pat-ip-failure",
			Condition:       Condition("[IP] == pat(10.*)"),
			Result:          &Result{IP: "255.255.255.255"},
			ExpectedSuccess: false,
			ExpectedOutput:  "[IP] (255.255.255.255) == pat(10.*)",
		},
		{
			Name:            "pat-status",
			Condition:       Condition("[STATUS] == pat(4*)"),
			Result:          &Result{HTTPStatus: 404},
			ExpectedSuccess: true,
			ExpectedOutput:  "[STATUS] == pat(4*)",
		},
		{
			Name:            "pat-status-failure",
			Condition:       Condition("[STATUS] == pat(4*)"),
			Result:          &Result{HTTPStatus: 200},
			ExpectedSuccess: false,
			ExpectedOutput:  "[STATUS] (200) == pat(4*)",
		},
		// any
		{
			Name:            "any-body-1",
			Condition:       Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:          &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].name == any(john.doe, jane.doe)",
		},
		{
			Name:            "any-body-2",
			Condition:       Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:          &Result{Body: []byte("{\"name\": \"jane.doe\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].name == any(john.doe, jane.doe)",
		},
		{
			Name:            "any-body-failure",
			Condition:       Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:          &Result{Body: []byte("{\"name\": \"bob\"}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].name (bob) == any(john.doe, jane.doe)",
		},
		{
			Name:            "any-status-1",
			Condition:       Condition("[STATUS] == any(200, 429)"),
			Result:          &Result{HTTPStatus: 200},
			ExpectedSuccess: true,
			ExpectedOutput:  "[STATUS] == any(200, 429)",
		},
		{
			Name:            "any-status-2",
			Condition:       Condition("[STATUS] == any(200, 429)"),
			Result:          &Result{HTTPStatus: 429},
			ExpectedSuccess: true,
			ExpectedOutput:  "[STATUS] == any(200, 429)",
		},
		{
			Name:            "any-status-reverse",
			Condition:       Condition("any(200, 429) == [STATUS]"),
			Result:          &Result{HTTPStatus: 429},
			ExpectedSuccess: true,
			ExpectedOutput:  "any(200, 429) == [STATUS]",
		},
		{
			Name:            "any-status-failure",
			Condition:       Condition("[STATUS] == any(200, 429)"),
			Result:          &Result{HTTPStatus: 404},
			ExpectedSuccess: false,
			ExpectedOutput:  "[STATUS] (404) == any(200, 429)",
		},
		{
			Name:                        "any-status-failure-but-dont-resolve",
			Condition:                   Condition("[STATUS] == any(200, 429)"),
			Result:                      &Result{HTTPStatus: 404},
			DontResolveFailedConditions: true,
			ExpectedSuccess:             false,
			ExpectedOutput:              "[STATUS] == any(200, 429)",
		},
		// has
		{
			Name:            "has",
			Condition:       Condition("has([BODY].errors) == false"),
			Result:          &Result{Body: []byte("{}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "has([BODY].errors) == false",
		},
		{
			Name:                        "has-key-of-map",
			Condition:                   Condition("has([BODY].article) == true"),
			Result:                      &Result{Body: []byte("{\n  \"article\": {\n    \"id\": 123,\n    \"title\": \"Hello, world!\",\n    \"author\": \"John Doe\",\n    \"tags\": [\"hello\", \"world\"],\n    \"content\": \"I really like Gatus!\"\n  }\n}")},
			DontResolveFailedConditions: false,
			ExpectedSuccess:             true,
			ExpectedOutput:              "has([BODY].article) == true",
		},
		{
			Name:            "has-failure",
			Condition:       Condition("has([BODY].errors) == false"),
			Result:          &Result{Body: []byte("{\"errors\": [\"1\"]}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "has([BODY].errors) (true) == false",
		},
		{
			Name:                        "has-failure-but-dont-resolve",
			Condition:                   Condition("has([BODY].errors) == false"),
			Result:                      &Result{Body: []byte("{\"errors\": [\"1\"]}")},
			DontResolveFailedConditions: true,
			ExpectedSuccess:             false,
			ExpectedOutput:              "has([BODY].errors) == false",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Condition.evaluate(scenario.Result, scenario.DontResolveFailedConditions)
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
	condition.evaluate(result, false)
	if result.Success {
		t.Error("condition was invalid, result should've been a failure")
	}
	if len(result.Errors) != 1 {
		t.Error("condition was invalid, result should've had an error")
	}
}

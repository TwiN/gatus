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
		ExpectedSeverity            SeverityStatus
	}{
		{
			Name:             "ip",
			Condition:        Condition("[IP] == 127.0.0.1"),
			Result:           &Result{IP: "127.0.0.1"},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[IP] == 127.0.0.1",
			ExpectedSeverity: None,
		},
		{
			Name:             "status",
			Condition:        Condition("[STATUS] == 200"),
			Result:           &Result{HTTPStatus: 200},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[STATUS] == 200",
			ExpectedSeverity: None,
		},
		{
			Name:             "status-failure",
			Condition:        Condition("[STATUS] == 200"),
			Result:           &Result{HTTPStatus: 500},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[STATUS] (500) == 200",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "status-failure",
			Condition:        Condition("Low :: [STATUS] == 200"),
			Result:           &Result{HTTPStatus: 500},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[STATUS] (500) == 200",
			ExpectedSeverity: Low,
		},
		{
			Name:             "status-failure",
			Condition:        Condition("Medium :: [STATUS] == 200"),
			Result:           &Result{HTTPStatus: 500},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[STATUS] (500) == 200",
			ExpectedSeverity: Medium,
		},
		{
			Name:             "status-failure",
			Condition:        Condition("High :: [STATUS] == 200"),
			Result:           &Result{HTTPStatus: 500},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[STATUS] (500) == 200",
			ExpectedSeverity: High,
		},
		{
			Name:             "status-failure",
			Condition:        Condition("Critical :: [STATUS] == 200"),
			Result:           &Result{HTTPStatus: 500},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[STATUS] (500) == 200",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "status-using-less-than",
			Condition:        Condition("[STATUS] < 300"),
			Result:           &Result{HTTPStatus: 201},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[STATUS] < 300",
			ExpectedSeverity: None,
		},
		{
			Name:             "status-using-less-than-failure",
			Condition:        Condition("[STATUS] < 300"),
			Result:           &Result{HTTPStatus: 404},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[STATUS] (404) < 300",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "response-time-using-less-than",
			Condition:        Condition("[RESPONSE_TIME] < 500"),
			Result:           &Result{Duration: 50 * time.Millisecond},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[RESPONSE_TIME] < 500",
			ExpectedSeverity: None,
		},
		{
			Name:             "response-time-using-less-than-with-duration",
			Condition:        Condition("[RESPONSE_TIME] < 1s"),
			Result:           &Result{Duration: 50 * time.Millisecond},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[RESPONSE_TIME] < 1s",
			ExpectedSeverity: None,
		},
		{
			Name:             "response-time-using-less-than-invalid",
			Condition:        Condition("[RESPONSE_TIME] < potato"),
			Result:           &Result{Duration: 50 * time.Millisecond},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[RESPONSE_TIME] (50) < potato (0)", // Non-numerical values automatically resolve to 0
			ExpectedSeverity: Critical,
		},
		{
			Name:             "response-time-using-greater-than",
			Condition:        Condition("[RESPONSE_TIME] > 500"),
			Result:           &Result{Duration: 750 * time.Millisecond},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[RESPONSE_TIME] > 500",
			ExpectedSeverity: None,
		},
		{
			Name:             "response-time-using-greater-than-with-duration",
			Condition:        Condition("[RESPONSE_TIME] > 1s"),
			Result:           &Result{Duration: 2 * time.Second},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[RESPONSE_TIME] > 1s",
			ExpectedSeverity: None,
		},
		{
			Name:             "response-time-using-greater-than-or-equal-to-equal",
			Condition:        Condition("[RESPONSE_TIME] >= 500"),
			Result:           &Result{Duration: 500 * time.Millisecond},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[RESPONSE_TIME] >= 500",
			ExpectedSeverity: None,
		},
		{
			Name:             "response-time-using-greater-than-or-equal-to-greater",
			Condition:        Condition("[RESPONSE_TIME] >= 500"),
			Result:           &Result{Duration: 499 * time.Millisecond},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[RESPONSE_TIME] (499) >= 500",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "response-time-using-greater-than-or-equal-to-failure",
			Condition:        Condition("[RESPONSE_TIME] >= 500"),
			Result:           &Result{Duration: 499 * time.Millisecond},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[RESPONSE_TIME] (499) >= 500",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "response-time-using-less-than-or-equal-to-equal",
			Condition:        Condition("[RESPONSE_TIME] <= 500"),
			Result:           &Result{Duration: 500 * time.Millisecond},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[RESPONSE_TIME] <= 500",
			ExpectedSeverity: None,
		},
		{
			Name:             "response-time-using-less-than-or-equal-to-less",
			Condition:        Condition("[RESPONSE_TIME] <= 500"),
			Result:           &Result{Duration: 25 * time.Millisecond},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[RESPONSE_TIME] <= 500",
			ExpectedSeverity: None,
		},
		{
			Name:             "response-time-using-less-than-or-equal-to-failure",
			Condition:        Condition("[RESPONSE_TIME] <= 500"),
			Result:           &Result{Duration: 750 * time.Millisecond},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[RESPONSE_TIME] (750) <= 500",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "body",
			Condition:        Condition("[BODY] == test"),
			Result:           &Result{Body: []byte("test")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY] == test",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-numerical-equal",
			Condition:        Condition("[BODY] == 123"),
			Result:           &Result{Body: []byte("123")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY] == 123",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-numerical-less-than",
			Condition:        Condition("[BODY] < 124"),
			Result:           &Result{Body: []byte("123")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY] < 124",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-numerical-greater-than",
			Condition:        Condition("[BODY] > 122"),
			Result:           &Result{Body: []byte("123")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY] > 122",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-numerical-greater-than-failure",
			Condition:        Condition("[BODY] > 123"),
			Result:           &Result{Body: []byte("100")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[BODY] (100) > 123",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "body-jsonpath",
			Condition:        Condition("[BODY].status == UP"),
			Result:           &Result{Body: []byte("{\"status\":\"UP\"}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].status == UP",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-jsonpath-complex",
			Condition:        Condition("[BODY].data.name == john"),
			Result:           &Result{Body: []byte("{\"data\": {\"id\": 1, \"name\": \"john\"}}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].data.name == john",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-jsonpath-complex-invalid",
			Condition:        Condition("[BODY].data.name == john"),
			Result:           &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[BODY].data.name (INVALID) == john",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "body-jsonpath-complex-len-invalid",
			Condition:        Condition("len([BODY].data.name) == john"),
			Result:           &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "len([BODY].data.name) (INVALID) == john",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "body-jsonpath-double-placeholder",
			Condition:        Condition("[BODY].user.firstName != [BODY].user.lastName"),
			Result:           &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].user.firstName != [BODY].user.lastName",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-jsonpath-double-placeholder-failure",
			Condition:        Condition("[BODY].user.firstName == [BODY].user.lastName"),
			Result:           &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[BODY].user.firstName (john) == [BODY].user.lastName (doe)",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "body-jsonpath-when-body-is-array",
			Condition:        Condition("[BODY][0].id == 1"),
			Result:           &Result{Body: []byte("[{\"id\": 1}, {\"id\": 2}]")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY][0].id == 1",
			ExpectedSeverity: None,
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-using-greater-than",
			Condition:       Condition("[BODY].data > 0"),
			Result:          &Result{Body: []byte("{\"data\": \"0x1\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data > 0",
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-using-equal-to-0x1",
			Condition:       Condition("[BODY].data == 1"),
			Result:          &Result{Body: []byte("{\"data\": \"0x1\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 1",
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-using-equals",
			Condition:       Condition("[BODY].data == 0x1"),
			Result:          &Result{Body: []byte("{\"data\": \"0x1\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 0x1",
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-using-equal-to-0x2",
			Condition:       Condition("[BODY].data == 2"),
			Result:          &Result{Body: []byte("{\"data\": \"0x2\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 2",
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-using-equal-to-0xF",
			Condition:       Condition("[BODY].data == 15"),
			Result:          &Result{Body: []byte("{\"data\": \"0xF\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 15",
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-using-equal-to-0xC0ff33",
			Condition:       Condition("[BODY].data == 12648243"),
			Result:          &Result{Body: []byte("{\"data\": \"0xC0ff33\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 12648243",
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-len",
			Condition:       Condition("len([BODY].data) == 3"),
			Result:          &Result{Body: []byte("{\"data\": \"0x1\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].data) == 3",
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-greater",
			Condition:       Condition("[BODY].data >= 1"),
			Result:          &Result{Body: []byte("{\"data\": \"0x01\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data >= 1",
		},
		{
			Name:            "body-jsonpath-hexadecimal-int-0x01-len",
			Condition:       Condition("len([BODY].data) == 4"),
			Result:          &Result{Body: []byte("{\"data\": \"0x01\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "len([BODY].data) == 4",
		},
		{
			Name:            "body-jsonpath-octal-int-using-greater-than",
			Condition:       Condition("[BODY].data > 0"),
			Result:          &Result{Body: []byte("{\"data\": \"0o1\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data > 0",
		},
		{
			Name:            "body-jsonpath-octal-int-using-equal",
			Condition:       Condition("[BODY].data == 2"),
			Result:          &Result{Body: []byte("{\"data\": \"0o2\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 2",
		},
		{
			Name:            "body-jsonpath-octal-int-using-equals",
			Condition:       Condition("[BODY].data == 0o2"),
			Result:          &Result{Body: []byte("{\"data\": \"0o2\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 0o2",
		},
		{
			Name:            "body-jsonpath-binary-int-using-greater-than",
			Condition:       Condition("[BODY].data > 0"),
			Result:          &Result{Body: []byte("{\"data\": \"0b1\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data > 0",
		},
		{
			Name:            "body-jsonpath-binary-int-using-equal",
			Condition:       Condition("[BODY].data == 2"),
			Result:          &Result{Body: []byte("{\"data\": \"0b0010\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 2",
		},
		{
			Name:            "body-jsonpath-binary-int-using-equals",
			Condition:       Condition("[BODY].data == 0b10"),
			Result:          &Result{Body: []byte("{\"data\": \"0b0010\"}")},
			ExpectedSuccess: true,
			ExpectedOutput:  "[BODY].data == 0b10",
		},
		{
			Name:            "body-jsonpath-complex-int-using-greater-than-failure",
			Condition:       Condition("[BODY].data.id > 5"),
			Result:          &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess: false,
			ExpectedOutput:  "[BODY].data.id (1) > 5",
		},
		{
			Name:             "body-jsonpath-complex-int",
			Condition:        Condition("[BODY].data.id == 1"),
			Result:           &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].data.id == 1",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-jsonpath-complex-array-int",
			Condition:        Condition("[BODY].data[1].id == 2"),
			Result:           &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].data[1].id == 2",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-jsonpath-complex-int-using-greater-than",
			Condition:        Condition("[BODY].data.id > 0"),
			Result:           &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].data.id > 0",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-jsonpath-complex-int-using-greater-than-failure",
			Condition:        Condition("[BODY].data.id > 5"),
			Result:           &Result{Body: []byte("{\"data\": {\"id\": 1}}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[BODY].data.id (1) > 5",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "body-jsonpath-float-using-greater-than-issue433", // As of v5.3.1, Gatus will convert a float to an int. We're losing precision, but it's better than just returning 0
			Condition:        Condition("[BODY].balance > 100"),
			Result:           &Result{Body: []byte(`{"balance": "123.40000000000005"}`)},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].balance > 100",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-jsonpath-complex-int-using-less-than",
			Condition:        Condition("[BODY].data.id < 5"),
			Result:           &Result{Body: []byte("{\"data\": {\"id\": 2}}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].data.id < 5",
			ExpectedSeverity: None,
		},
		{
			Name:             "body-jsonpath-complex-int-using-less-than-failure",
			Condition:        Condition("[BODY].data.id < 5"),
			Result:           &Result{Body: []byte("{\"data\": {\"id\": 10}}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[BODY].data.id (10) < 5",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "connected",
			Condition:        Condition("[CONNECTED] == true"),
			Result:           &Result{Connected: true},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[CONNECTED] == true",
			ExpectedSeverity: None,
		},
		{
			Name:             "connected-failure",
			Condition:        Condition("[CONNECTED] == true"),
			Result:           &Result{Connected: false},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[CONNECTED] (false) == true",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "certificate-expiration-not-set",
			Condition:        Condition("[CERTIFICATE_EXPIRATION] == 0"),
			Result:           &Result{},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[CERTIFICATE_EXPIRATION] == 0",
			ExpectedSeverity: None,
		},
		{
			Name:             "certificate-expiration-greater-than-numerical",
			Condition:        Condition("[CERTIFICATE_EXPIRATION] > " + strconv.FormatInt((time.Hour*24*28).Milliseconds(), 10)),
			Result:           &Result{CertificateExpiration: time.Hour * 24 * 60},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[CERTIFICATE_EXPIRATION] > 2419200000",
			ExpectedSeverity: None,
		},
		{
			Name:             "certificate-expiration-greater-than-numerical-failure",
			Condition:        Condition("[CERTIFICATE_EXPIRATION] > " + strconv.FormatInt((time.Hour*24*28).Milliseconds(), 10)),
			Result:           &Result{CertificateExpiration: time.Hour * 24 * 14},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[CERTIFICATE_EXPIRATION] (1209600000) > 2419200000",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "certificate-expiration-greater-than-duration",
			Condition:        Condition("[CERTIFICATE_EXPIRATION] > 12h"),
			Result:           &Result{CertificateExpiration: 24 * time.Hour},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[CERTIFICATE_EXPIRATION] > 12h",
			ExpectedSeverity: None,
		},
		{
			Name:             "certificate-expiration-greater-than-duration",
			Condition:        Condition("[CERTIFICATE_EXPIRATION] > 48h"),
			Result:           &Result{CertificateExpiration: 24 * time.Hour},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[CERTIFICATE_EXPIRATION] (86400000) > 48h (172800000)",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "no-placeholders",
			Condition:        Condition("1 == 2"),
			Result:           &Result{},
			ExpectedSuccess:  false,
			ExpectedOutput:   "1 == 2",
			ExpectedSeverity: Critical,
		},
		///////////////
		// Functions //
		///////////////
		// len
		{
			Name:             "len-body-jsonpath-complex",
			Condition:        Condition("len([BODY].data.name) == 4"),
			Result:           &Result{Body: []byte("{\"data\": {\"name\": \"john\"}}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY].data.name) == 4",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-array",
			Condition:        Condition("len([BODY]) == 3"),
			Result:           &Result{Body: []byte("[{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY]) == 3",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-keyed-array",
			Condition:        Condition("len([BODY].data) == 3"),
			Result:           &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY].data) == 3",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-array-invalid",
			Condition:        Condition("len([BODY].data) == 8"),
			Result:           &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "len([BODY].data) (INVALID) == 8",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "len-body-string",
			Condition:        Condition("len([BODY]) == 8"),
			Result:           &Result{Body: []byte("john.doe")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY]) == 8",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-keyed-string",
			Condition:        Condition("len([BODY].name) == 8"),
			Result:           &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY].name) == 8",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-keyed-int",
			Condition:        Condition("len([BODY].age) == 2"),
			Result:           &Result{Body: []byte(`{"age":18}`)},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY].age) == 2",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-keyed-bool",
			Condition:        Condition("len([BODY].adult) == 4"),
			Result:           &Result{Body: []byte(`{"adult":true}`)},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY].adult) == 4",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-object-inside-array",
			Condition:        Condition("len([BODY][0]) == 23"),
			Result:           &Result{Body: []byte(`[{"age":18,"adult":true}]`)},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY][0]) == 23",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-object-keyed-int-inside-array",
			Condition:        Condition("len([BODY][0].age) == 2"),
			Result:           &Result{Body: []byte(`[{"age":18,"adult":true}]`)},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY][0].age) == 2",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-keyed-bool-inside-array",
			Condition:        Condition("len([BODY][0].adult) == 4"),
			Result:           &Result{Body: []byte(`[{"age":18,"adult":true}]`)},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY][0].adult) == 4",
			ExpectedSeverity: None,
		},
		{
			Name:             "len-body-object",
			Condition:        Condition("len([BODY]) == 20"),
			Result:           &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "len([BODY]) == 20",
			ExpectedSeverity: None,
		},
		// pat
		{
			Name:             "pat-body-1",
			Condition:        Condition("[BODY] == pat(*john*)"),
			Result:           &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY] == pat(*john*)",
			ExpectedSeverity: None,
		},
		{
			Name:             "pat-body-2",
			Condition:        Condition("[BODY].name == pat(john*)"),
			Result:           &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].name == pat(john*)",
			ExpectedSeverity: None,
		},
		{
			Name:             "pat-body-failure",
			Condition:        Condition("[BODY].name == pat(bob*)"),
			Result:           &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[BODY].name (john.doe) == pat(bob*)",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "pat-body-html",
			Condition:        Condition("[BODY] == pat(*<div id=\"user\">john.doe</div>*)"),
			Result:           &Result{Body: []byte(`<!DOCTYPE html><html lang="en"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head><body><div id="user">john.doe</div></body></html>`)},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY] == pat(*<div id=\"user\">john.doe</div>*)",
			ExpectedSeverity: None,
		},
		{
			Name:             "pat-body-html-failure",
			Condition:        Condition("[BODY] == pat(*<div id=\"user\">john.doe</div>*)"),
			Result:           &Result{Body: []byte(`<!DOCTYPE html><html lang="en"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head><body><div id="user">jane.doe</div></body></html>`)},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[BODY] (<!DOCTYPE html><html lang...(truncated)) == pat(*<div id=\"user\">john.doe</div>*)",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "pat-body-html-failure-alt",
			Condition:        Condition("pat(*<div id=\"user\">john.doe</div>*) == [BODY]"),
			Result:           &Result{Body: []byte(`<!DOCTYPE html><html lang="en"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head><body><div id="user">jane.doe</div></body></html>`)},
			ExpectedSuccess:  false,
			ExpectedOutput:   "pat(*<div id=\"user\">john.doe</div>*) == [BODY] (<!DOCTYPE html><html lang...(truncated))",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "pat-body-in-array",
			Condition:        Condition("[BODY].data == pat(*Whatever*)"),
			Result:           &Result{Body: []byte("{\"data\": [\"hello\", \"world\", \"Whatever\"]}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].data == pat(*Whatever*)",
			ExpectedSeverity: None,
		},
		{
			Name:             "pat-ip",
			Condition:        Condition("[IP] == pat(10.*)"),
			Result:           &Result{IP: "10.0.0.0"},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[IP] == pat(10.*)",
			ExpectedSeverity: None,
		},
		{
			Name:             "pat-ip-failure",
			Condition:        Condition("[IP] == pat(10.*)"),
			Result:           &Result{IP: "255.255.255.255"},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[IP] (255.255.255.255) == pat(10.*)",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "pat-status",
			Condition:        Condition("[STATUS] == pat(4*)"),
			Result:           &Result{HTTPStatus: 404},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[STATUS] == pat(4*)",
			ExpectedSeverity: None,
		},
		{
			Name:             "pat-status-failure",
			Condition:        Condition("[STATUS] == pat(4*)"),
			Result:           &Result{HTTPStatus: 200},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[STATUS] (200) == pat(4*)",
			ExpectedSeverity: Critical,
		},
		// any
		{
			Name:             "any-body-1",
			Condition:        Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:           &Result{Body: []byte("{\"name\": \"john.doe\"}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].name == any(john.doe, jane.doe)",
			ExpectedSeverity: None,
		},
		{
			Name:             "any-body-2",
			Condition:        Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:           &Result{Body: []byte("{\"name\": \"jane.doe\"}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[BODY].name == any(john.doe, jane.doe)",
			ExpectedSeverity: None,
		},
		{
			Name:             "any-body-failure",
			Condition:        Condition("[BODY].name == any(john.doe, jane.doe)"),
			Result:           &Result{Body: []byte("{\"name\": \"bob\"}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[BODY].name (bob) == any(john.doe, jane.doe)",
			ExpectedSeverity: Critical,
		},
		{
			Name:             "any-status-1",
			Condition:        Condition("[STATUS] == any(200, 429)"),
			Result:           &Result{HTTPStatus: 200},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[STATUS] == any(200, 429)",
			ExpectedSeverity: None,
		},
		{
			Name:             "any-status-2",
			Condition:        Condition("[STATUS] == any(200, 429)"),
			Result:           &Result{HTTPStatus: 429},
			ExpectedSuccess:  true,
			ExpectedOutput:   "[STATUS] == any(200, 429)",
			ExpectedSeverity: None,
		},
		{
			Name:             "any-status-reverse",
			Condition:        Condition("any(200, 429) == [STATUS]"),
			Result:           &Result{HTTPStatus: 429},
			ExpectedSuccess:  true,
			ExpectedOutput:   "any(200, 429) == [STATUS]",
			ExpectedSeverity: None,
		},
		{
			Name:             "any-status-failure",
			Condition:        Condition("[STATUS] == any(200, 429)"),
			Result:           &Result{HTTPStatus: 404},
			ExpectedSuccess:  false,
			ExpectedOutput:   "[STATUS] (404) == any(200, 429)",
			ExpectedSeverity: Critical,
		},
		{
			Name:                        "any-status-failure-but-dont-resolve",
			Condition:                   Condition("[STATUS] == any(200, 429)"),
			Result:                      &Result{HTTPStatus: 404},
			DontResolveFailedConditions: true,
			ExpectedSuccess:             false,
			ExpectedOutput:              "[STATUS] == any(200, 429)",
			ExpectedSeverity:            Critical,
		},
		// has
		{
			Name:             "has",
			Condition:        Condition("has([BODY].errors) == false"),
			Result:           &Result{Body: []byte("{}")},
			ExpectedSuccess:  true,
			ExpectedOutput:   "has([BODY].errors) == false",
			ExpectedSeverity: None,
		},
		{
			Name:                        "has-key-of-map",
			Condition:                   Condition("has([BODY].article) == true"),
			Result:                      &Result{Body: []byte("{\n  \"article\": {\n    \"id\": 123,\n    \"title\": \"Hello, world!\",\n    \"author\": \"John Doe\",\n    \"tags\": [\"hello\", \"world\"],\n    \"content\": \"I really like Gatus!\"\n  }\n}")},
			DontResolveFailedConditions: false,
			ExpectedSuccess:             true,
			ExpectedOutput:              "has([BODY].article) == true",
			ExpectedSeverity:            None,
		},
		{
			Name:             "has-failure",
			Condition:        Condition("has([BODY].errors) == false"),
			Result:           &Result{Body: []byte("{\"errors\": [\"1\"]}")},
			ExpectedSuccess:  false,
			ExpectedOutput:   "has([BODY].errors) (true) == false",
			ExpectedSeverity: Critical,
		},
		{
			Name:                        "has-failure-but-dont-resolve",
			Condition:                   Condition("has([BODY].errors) == false"),
			Result:                      &Result{Body: []byte("{\"errors\": [\"1\"]}")},
			DontResolveFailedConditions: true,
			ExpectedSuccess:             false,
			ExpectedOutput:              "has([BODY].errors) == false",
			ExpectedSeverity:            Critical,
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
			if scenario.Result.ConditionResults[0].SeverityStatus != scenario.ExpectedSeverity {
				t.Errorf("Severity '%s' should have resolved to '%v', got '%v'", scenario.Condition, scenario.ExpectedSeverity, scenario.Result.ConditionResults[0].SeverityStatus)
			}
		})
	}
}

func TestCondition_sanitizeSeverityCondition(t *testing.T) {
	scenarios := []struct {
		condition         Condition
		expectedSeverity  SeverityStatus
		expectedCondition string
	}{
		{condition: "[STATUS] == 200", expectedSeverity: Critical, expectedCondition: "[STATUS] == 200"},
		{condition: "Low :: [STATUS] == 201", expectedSeverity: Low, expectedCondition: "[STATUS] == 201"},
		{condition: "Medium :: [STATUS] == 404", expectedSeverity: Medium, expectedCondition: "[STATUS] == 404"},
		{condition: "High :: [STATUS] == 500", expectedSeverity: High, expectedCondition: "[STATUS] == 500"},
		{condition: "Critical :: [STATUS] == 500", expectedSeverity: Critical, expectedCondition: "[STATUS] == 500"},
		{condition: "NotValid :: [STATUS] == 42", expectedSeverity: Critical, expectedCondition: "NotValid :: [STATUS] == 42"},
	}

	for _, scenario := range scenarios {
		t.Run(string(scenario.condition), func(t *testing.T) {
			severity, condition := scenario.condition.sanitizeSeverityCondition()
			if fmt.Sprint(severity) != fmt.Sprint(scenario.expectedSeverity) {
				t.Errorf("expected %v, got %v", scenario.expectedSeverity, severity)
			}
			if fmt.Sprint(condition) != fmt.Sprint(scenario.expectedCondition) {
				t.Errorf("expected %v, got %v", scenario.expectedCondition, condition)
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

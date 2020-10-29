package core

import (
	"testing"
	"time"
)

func TestCondition_evaluateWithIP(t *testing.T) {
	condition := Condition("[IP] == 127.0.0.1")
	result := &Result{IP: "127.0.0.1"}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithStatus(t *testing.T) {
	condition := Condition("[STATUS] == 201")
	result := &Result{HTTPStatus: 201}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithStatusFailure(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	result := &Result{HTTPStatus: 500}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithStatusUsingLessThan(t *testing.T) {
	condition := Condition("[STATUS] < 300")
	result := &Result{HTTPStatus: 201}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithStatusFailureUsingLessThan(t *testing.T) {
	condition := Condition("[STATUS] < 300")
	result := &Result{HTTPStatus: 404}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingLessThan(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] < 500")
	result := &Result{Duration: time.Millisecond * 50}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingGreaterThan(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] > 500")
	result := &Result{Duration: time.Millisecond * 750}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingGreaterThanOrEqualTo(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] >= 500")
	result := &Result{Duration: time.Millisecond * 500}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingLessThanOrEqualTo(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] <= 500")
	result := &Result{Duration: time.Millisecond * 500}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBody(t *testing.T) {
	condition := Condition("[BODY] == test")
	result := &Result{Body: []byte("test")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPath(t *testing.T) {
	condition := Condition("[BODY].status == UP")
	result := &Result{Body: []byte("{\"status\":\"UP\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplex(t *testing.T) {
	condition := Condition("[BODY].data.name == john")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1, \"name\": \"john\"}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithInvalidBodyJSONPathComplex(t *testing.T) {
	expectedResolvedCondition := "[BODY].data.name (INVALID) == john"
	condition := Condition("[BODY].data.name == john")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure, because the path was invalid", condition)
	}
	if result.ConditionResults[0].Condition != expectedResolvedCondition {
		t.Errorf("Condition '%s' should have resolved to '%s', but resolved to '%s' instead", condition, expectedResolvedCondition, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithInvalidBodyJSONPathComplexWithLengthFunction(t *testing.T) {
	expectedResolvedCondition := "len([BODY].data.name) (INVALID) == john"
	condition := Condition("len([BODY].data.name) == john")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure, because the path was invalid", condition)
	}
	if result.ConditionResults[0].Condition != expectedResolvedCondition {
		t.Errorf("Condition '%s' should have resolved to '%s', but resolved to '%s' instead", condition, expectedResolvedCondition, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathDoublePlaceholders(t *testing.T) {
	condition := Condition("[BODY].user.firstName != [BODY].user.lastName")
	result := &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathDoublePlaceholdersFailure(t *testing.T) {
	condition := Condition("[BODY].user.firstName == [BODY].user.lastName")
	result := &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathLongInt(t *testing.T) {
	condition := Condition("[BODY].data.id == 1")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexInt(t *testing.T) {
	condition := Condition("[BODY].data[1].id == 2")
	result := &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexIntUsingGreaterThan(t *testing.T) {
	condition := Condition("[BODY].data.id > 0")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexIntFailureUsingGreaterThan(t *testing.T) {
	condition := Condition("[BODY].data.id > 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexIntUsingLessThan(t *testing.T) {
	condition := Condition("[BODY].data.id < 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 2}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexIntFailureUsingLessThan(t *testing.T) {
	condition := Condition("[BODY].data.id < 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 10}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithBodySliceLength(t *testing.T) {
	condition := Condition("len([BODY].data) == 3")
	result := &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyStringLength(t *testing.T) {
	condition := Condition("len([BODY].name) == 8")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyStringPattern(t *testing.T) {
	condition := Condition("[BODY].name == pat(*ohn*)")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyStringPatternFailure(t *testing.T) {
	condition := Condition("[BODY].name == pat(bob*)")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithBodyPatternFailure(t *testing.T) {
	condition := Condition("[BODY] == pat(*john*)")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithIPPattern(t *testing.T) {
	condition := Condition("[IP] == pat(10.*)")
	result := &Result{IP: "10.0.0.0"}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithIPPatternFailure(t *testing.T) {
	condition := Condition("[IP] == pat(10.*)")
	result := &Result{IP: "255.255.255.255"}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithStatusPattern(t *testing.T) {
	condition := Condition("[STATUS] == pat(4*)")
	result := &Result{HTTPStatus: 404}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithStatusPatternFailure(t *testing.T) {
	condition := Condition("[STATUS] != pat(4*)")
	result := &Result{HTTPStatus: 404}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithConnected(t *testing.T) {
	condition := Condition("[CONNECTED] == true")
	result := &Result{Connected: true}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithConnectedFailure(t *testing.T) {
	condition := Condition("[CONNECTED] == true")
	result := &Result{Connected: false}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

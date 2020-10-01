package core

import (
	"testing"
	"time"
)

func TestCondition_evaluateWithIp(t *testing.T) {
	condition := Condition("[IP] == 127.0.0.1")
	result := &Result{Ip: "127.0.0.1"}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithStatus(t *testing.T) {
	condition := Condition("[STATUS] == 201")
	result := &Result{HttpStatus: 201}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithStatusFailure(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	result := &Result{HttpStatus: 500}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithStatusUsingLessThan(t *testing.T) {
	condition := Condition("[STATUS] < 300")
	result := &Result{HttpStatus: 201}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithStatusFailureUsingLessThan(t *testing.T) {
	condition := Condition("[STATUS] < 300")
	result := &Result{HttpStatus: 404}
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

func TestCondition_evaluateWithBodyJsonPath(t *testing.T) {
	condition := Condition("[BODY].status == UP")
	result := &Result{Body: []byte("{\"status\":\"UP\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathComplex(t *testing.T) {
	condition := Condition("[BODY].data.name == john")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1, \"name\": \"john\"}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathDoublePlaceholders(t *testing.T) {
	condition := Condition("[BODY].user.firstName != [BODY].user.lastName")
	result := &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathDoublePlaceholdersFailure(t *testing.T) {
	condition := Condition("[BODY].user.firstName == [BODY].user.lastName")
	result := &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathLongInt(t *testing.T) {
	condition := Condition("[BODY].data.id == 1")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathComplexInt(t *testing.T) {
	condition := Condition("[BODY].data[1].id == 2")
	result := &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathComplexIntUsingGreaterThan(t *testing.T) {
	condition := Condition("[BODY].data.id > 0")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathComplexIntFailureUsingGreaterThan(t *testing.T) {
	condition := Condition("[BODY].data.id > 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathComplexIntUsingLessThan(t *testing.T) {
	condition := Condition("[BODY].data.id < 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 2}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithBodyJsonPathComplexIntFailureUsingLessThan(t *testing.T) {
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
	result := &Result{Ip: "10.0.0.0"}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithIPPatternFailure(t *testing.T) {
	condition := Condition("[IP] == pat(10.*)")
	result := &Result{Ip: "255.255.255.255"}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestCondition_evaluateWithStatusPattern(t *testing.T) {
	condition := Condition("[STATUS] == pat(4*)")
	result := &Result{HttpStatus: 404}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestCondition_evaluateWithStatusPatternFailure(t *testing.T) {
	condition := Condition("[STATUS] != pat(4*)")
	result := &Result{HttpStatus: 404}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

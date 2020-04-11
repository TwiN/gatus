package core

import (
	"testing"
	"time"
)

func TestEvaluateWithIp(t *testing.T) {
	condition := Condition("[IP] == 127.0.0.1")
	result := &Result{Ip: "127.0.0.1"}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithStatus(t *testing.T) {
	condition := Condition("[STATUS] == 201")
	result := &Result{HttpStatus: 201}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithStatusFailure(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	result := &Result{HttpStatus: 500}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestEvaluateWithStatusUsingLessThan(t *testing.T) {
	condition := Condition("[STATUS] < 300")
	result := &Result{HttpStatus: 201}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithStatusFailureUsingLessThan(t *testing.T) {
	condition := Condition("[STATUS] < 300")
	result := &Result{HttpStatus: 404}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestEvaluateWithResponseTimeUsingLessThan(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] < 500")
	result := &Result{Duration: time.Millisecond * 50}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithResponseTimeUsingGreaterThan(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] > 500")
	result := &Result{Duration: time.Millisecond * 750}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithResponseTimeUsingGreaterThanOrEqualTo(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] >= 500")
	result := &Result{Duration: time.Millisecond * 500}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithResponseTimeUsingLessThanOrEqualTo(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] <= 500")
	result := &Result{Duration: time.Millisecond * 500}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithBody(t *testing.T) {
	condition := Condition("[BODY] == test")
	result := &Result{Body: []byte("test")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithBodyJsonPath(t *testing.T) {
	condition := Condition("[BODY].status == UP")
	result := &Result{Body: []byte("{\"status\":\"UP\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithBodyJsonPathComplex(t *testing.T) {
	condition := Condition("[BODY].data.name == john")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1, \"name\": \"john\"}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithBodyJsonPathComplexInt(t *testing.T) {
	condition := Condition("[BODY].data.id == 1")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithBodyJsonPathComplexIntUsingGreaterThan(t *testing.T) {
	condition := Condition("[BODY].data.id > 0")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithBodyJsonPathComplexIntFailureUsingGreaterThan(t *testing.T) {
	condition := Condition("[BODY].data.id > 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestEvaluateWithBodyJsonPathComplexIntUsingLessThan(t *testing.T) {
	condition := Condition("[BODY].data.id < 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 2}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithBodyJsonPathComplexIntFailureUsingLessThan(t *testing.T) {
	condition := Condition("[BODY].data.id < 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 10}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
}

func TestIntegrationEvaluateConditions(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		Url:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
	}
	result := service.EvaluateConditions()
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	if !result.Success {
		t.Error("Because all conditions passed, this should have been a success")
	}
}

func TestIntegrationEvaluateConditionsWithFailure(t *testing.T) {
	condition := Condition("[STATUS] == 500")
	service := Service{
		Name:       "TwiNNatioN",
		Url:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
	}
	result := service.EvaluateConditions()
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	if result.Success {
		t.Error("Because one of the conditions failed, success should have been false")
	}
}

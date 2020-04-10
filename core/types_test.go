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

func TestIntegrationEvaluateConditions(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "GitHub",
		Url:        "https://api.github.com/healthz",
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
		Name:       "GitHub",
		Url:        "https://api.github.com/healthz",
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

package core

import (
	"testing"
)

func TestEvaluateWithIp(t *testing.T) {
	condition := Condition("[IP] == 127.0.0.1")
	result := &Result{Ip: "127.0.0.1"}
	condition.Evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithStatus(t *testing.T) {
	condition := Condition("[STATUS] == 201")
	result := &Result{HttpStatus: 201}
	condition.Evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
}

func TestEvaluateWithFailure(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	result := &Result{HttpStatus: 500}
	condition.Evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
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

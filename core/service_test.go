package core

import (
	"testing"
)

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

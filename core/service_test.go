package core

import (
	"testing"
)

func TestIntegrationEvaluateHealth(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		Url:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
	}
	result := service.EvaluateHealth()
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	if !result.Success {
		t.Error("Because all conditions passed, this should have been a success")
	}
}

func TestIntegrationEvaluateHealthWithFailure(t *testing.T) {
	condition := Condition("[STATUS] == 500")
	service := Service{
		Name:       "TwiNNatioN",
		Url:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
	}
	result := service.EvaluateHealth()
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	if result.Success {
		t.Error("Because one of the conditions failed, success should have been false")
	}
}

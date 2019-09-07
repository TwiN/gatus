package core

import (
	"testing"
)

func TestEvaluateWithIp(t *testing.T) {
	condition := Condition("$IP == 127.0.0.1")
	result := &Result{Ip: "127.0.0.1"}
	condition.Evaluate(result)
	if result.ConditionResult[0].Success != true {
		t.Error("Condition '$IP == 127.0.0.1' should have been a success")
	}
}

func TestEvaluateWithStatus(t *testing.T) {
	condition := Condition("$STATUS == 201")
	result := &Result{HttpStatus: 201}
	condition.Evaluate(result)
	if result.ConditionResult[0].Success != true {
		t.Error("Condition '$STATUS == 201' should have been a success")
	}
}

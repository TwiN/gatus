package core

import (
	"fmt"
	"strconv"
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
	if result.ConditionResults[0].Condition != string(condition) {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, condition, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithStatusFailure(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	result := &Result{HTTPStatus: 500}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[STATUS] (500) == 200"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithStatusUsingLessThan(t *testing.T) {
	condition := Condition("[STATUS] < 300")
	result := &Result{HTTPStatus: 201}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[STATUS] < 300"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithStatusFailureUsingLessThan(t *testing.T) {
	condition := Condition("[STATUS] < 300")
	result := &Result{HTTPStatus: 404}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[STATUS] (404) < 300"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingLessThan(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] < 500")
	result := &Result{Duration: time.Millisecond * 50}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[RESPONSE_TIME] < 500"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingLessThanDuration(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] < 1s")
	result := &Result{Duration: time.Millisecond * 50}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[RESPONSE_TIME] < 1s"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingLessThanInvalid(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] < potato")
	result := &Result{Duration: time.Millisecond * 50}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have failed because the condition has an invalid numerical value that should've automatically resolved to 0", condition)
	}
	expectedConditionDisplayed := "[RESPONSE_TIME] (50) < potato (0)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingGreaterThan(t *testing.T) {
	// Not exactly sure why you'd want to have a condition that checks if the response time is too fast,
	// but hey, who am I to judge?
	condition := Condition("[RESPONSE_TIME] > 500")
	result := &Result{Duration: time.Millisecond * 750}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[RESPONSE_TIME] > 500"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingGreaterThanDuration(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] > 1s")
	result := &Result{Duration: time.Second * 2}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[RESPONSE_TIME] > 1s"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingGreaterThanOrEqualTo(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] >= 500")
	result := &Result{Duration: time.Millisecond * 500}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[RESPONSE_TIME] >= 500"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithResponseTimeUsingLessThanOrEqualTo(t *testing.T) {
	condition := Condition("[RESPONSE_TIME] <= 500")
	result := &Result{Duration: time.Millisecond * 500}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[RESPONSE_TIME] <= 500"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBody(t *testing.T) {
	condition := Condition("[BODY] == test")
	result := &Result{Body: []byte("test")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY] == test"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPath(t *testing.T) {
	condition := Condition("[BODY].status == UP")
	result := &Result{Body: []byte("{\"status\":\"UP\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY].status == UP"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplex(t *testing.T) {
	condition := Condition("[BODY].data.name == john")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1, \"name\": \"john\"}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY].data.name == john"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithInvalidBodyJSONPathComplex(t *testing.T) {
	condition := Condition("[BODY].data.name == john")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure, because the path was invalid", condition)
	}
	expectedConditionDisplayed := "[BODY].data.name (INVALID) == john"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithInvalidBodyJSONPathComplexWithLengthFunction(t *testing.T) {
	condition := Condition("len([BODY].data.name) == john")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure, because the path was invalid", condition)
	}
	expectedConditionDisplayed := "len([BODY].data.name) (INVALID) == john"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathDoublePlaceholders(t *testing.T) {
	condition := Condition("[BODY].user.firstName != [BODY].user.lastName")
	result := &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY].user.firstName != [BODY].user.lastName"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathDoublePlaceholdersFailure(t *testing.T) {
	condition := Condition("[BODY].user.firstName == [BODY].user.lastName")
	result := &Result{Body: []byte("{\"user\": {\"firstName\": \"john\", \"lastName\": \"doe\"}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[BODY].user.firstName (john) == [BODY].user.lastName (doe)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathLongInt(t *testing.T) {
	condition := Condition("[BODY].data.id == 1")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY].data.id == 1"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexInt(t *testing.T) {
	condition := Condition("[BODY].data[1].id == 2")
	result := &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY].data[1].id == 2"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexIntUsingGreaterThan(t *testing.T) {
	condition := Condition("[BODY].data.id > 0")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY].data.id > 0"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexIntFailureUsingGreaterThan(t *testing.T) {
	condition := Condition("[BODY].data.id > 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 1}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[BODY].data.id (1) > 5"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexIntUsingLessThan(t *testing.T) {
	condition := Condition("[BODY].data.id < 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 2}}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY].data.id < 5"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyJSONPathComplexIntFailureUsingLessThan(t *testing.T) {
	condition := Condition("[BODY].data.id < 5")
	result := &Result{Body: []byte("{\"data\": {\"id\": 10}}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[BODY].data.id (10) < 5"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodySliceLength(t *testing.T) {
	condition := Condition("len([BODY].data) == 3")
	result := &Result{Body: []byte("{\"data\": [{\"id\": 1}, {\"id\": 2}, {\"id\": 3}]}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "len([BODY].data) == 3"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyStringLength(t *testing.T) {
	condition := Condition("len([BODY].name) == 8")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "len([BODY].name) == 8"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyPattern(t *testing.T) {
	condition := Condition("[BODY] == pat(*john*)")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY] == pat(*john*)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithReverseBodyPattern(t *testing.T) {
	condition := Condition("pat(*john*) == [BODY]")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "pat(*john*) == [BODY]"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyStringPattern(t *testing.T) {
	condition := Condition("[BODY].name == pat(*ohn*)")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY].name == pat(*ohn*)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyHTMLPattern(t *testing.T) {
	var html = `<!DOCTYPE html><html lang="en"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head><body><div id="user">john.doe</div></body></html>`
	condition := Condition("[BODY] == pat(*<div id=\"user\">john.doe</div>*)")
	result := &Result{Body: []byte(html)}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[BODY] == pat(*<div id=\"user\">john.doe</div>*)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyStringPatternFailure(t *testing.T) {
	condition := Condition("[BODY].name == pat(bob*)")
	result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[BODY].name (john.doe) == pat(bob*)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithIPPattern(t *testing.T) {
	condition := Condition("[IP] == pat(10.*)")
	result := &Result{IP: "10.0.0.0"}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[IP] == pat(10.*)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithIPPatternFailure(t *testing.T) {
	condition := Condition("[IP] == pat(10.*)")
	result := &Result{IP: "255.255.255.255"}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[IP] (255.255.255.255) == pat(10.*)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithStatusPattern(t *testing.T) {
	condition := Condition("[STATUS] == pat(4*)")
	result := &Result{HTTPStatus: 404}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[STATUS] == pat(4*)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithStatusPatternFailure(t *testing.T) {
	condition := Condition("[STATUS] != pat(4*)")
	result := &Result{HTTPStatus: 404}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[STATUS] (404) != pat(4*)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithBodyStringAny(t *testing.T) {
	condition := Condition("[BODY].name == any(john.doe, jane.doe)")
	expectedConditionDisplayed := "[BODY].name == any(john.doe, jane.doe)"
	results := []*Result{
		{Body: []byte("{\"name\": \"john.doe\"}")},
		{Body: []byte("{\"name\": \"jane.doe\"}")},
	}
	for _, result := range results {
		success := condition.evaluate(result)
		if !success || !result.ConditionResults[0].Success {
			t.Errorf("Condition '%s' should have been a success", condition)
		}
		if result.ConditionResults[0].Condition != expectedConditionDisplayed {
			t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
		}
	}
}

func TestCondition_evaluateWithBodyStringAnyFailure(t *testing.T) {
	condition := Condition("[BODY].name == any(john.doe, jane.doe)")
	result := &Result{Body: []byte("{\"name\": \"bob.doe\"}")}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[BODY].name (bob.doe) == any(john.doe, jane.doe)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithStatusAny(t *testing.T) {
	condition := Condition("[STATUS] == any(200, 429)")
	statuses := []int{200, 429}
	for _, status := range statuses {
		result := &Result{HTTPStatus: status}
		condition.evaluate(result)
		if !result.ConditionResults[0].Success {
			t.Errorf("Condition '%s' should have been a success", condition)
		}
		expectedConditionDisplayed := "[STATUS] == any(200, 429)"
		if result.ConditionResults[0].Condition != expectedConditionDisplayed {
			t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
		}
	}
}

func TestCondition_evaluateWithReverseStatusAny(t *testing.T) {
	condition := Condition("any(200, 429) == [STATUS]")
	statuses := []int{200, 429}
	for _, status := range statuses {
		result := &Result{HTTPStatus: status}
		condition.evaluate(result)
		if !result.ConditionResults[0].Success {
			t.Errorf("Condition '%s' should have been a success", condition)
		}
		expectedConditionDisplayed := "any(200, 429) == [STATUS]"
		if result.ConditionResults[0].Condition != expectedConditionDisplayed {
			t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
		}
	}
}

func TestCondition_evaluateWithStatusAnyFailure(t *testing.T) {
	condition := Condition("[STATUS] == any(200, 429)")
	statuses := []int{201, 400, 404, 500}
	for _, status := range statuses {
		result := &Result{HTTPStatus: status}
		condition.evaluate(result)
		if result.ConditionResults[0].Success {
			t.Errorf("Condition '%s' should have been a failure", condition)
		}
		expectedConditionDisplayed := fmt.Sprintf("[STATUS] (%d) == any(200, 429)", status)
		if result.ConditionResults[0].Condition != expectedConditionDisplayed {
			t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
		}
	}
}

func TestCondition_evaluateWithReverseStatusAnyFailure(t *testing.T) {
	condition := Condition("any(200, 429) == [STATUS]")
	statuses := []int{201, 400, 404, 500}
	for _, status := range statuses {
		result := &Result{HTTPStatus: status}
		condition.evaluate(result)
		if result.ConditionResults[0].Success {
			t.Errorf("Condition '%s' should have been a failure", condition)
		}
		expectedConditionDisplayed := fmt.Sprintf("any(200, 429) == [STATUS] (%d)", status)
		if result.ConditionResults[0].Condition != expectedConditionDisplayed {
			t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
		}
	}
}

func TestCondition_evaluateWithConnected(t *testing.T) {
	condition := Condition("[CONNECTED] == true")
	result := &Result{Connected: true}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[CONNECTED] == true"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithConnectedFailure(t *testing.T) {
	condition := Condition("[CONNECTED] == true")
	result := &Result{Connected: false}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[CONNECTED] (false) == true"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithUnsetCertificateExpiration(t *testing.T) {
	condition := Condition("[CERTIFICATE_EXPIRATION] == 0")
	result := &Result{}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[CERTIFICATE_EXPIRATION] == 0"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithCertificateExpirationGreaterThanNumerical(t *testing.T) {
	acceptable := (time.Hour * 24 * 28).Milliseconds()
	condition := Condition("[CERTIFICATE_EXPIRATION] > " + strconv.FormatInt(acceptable, 10))
	result := &Result{CertificateExpiration: time.Hour * 24 * 60}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[CERTIFICATE_EXPIRATION] > 2419200000"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithCertificateExpirationGreaterThanNumericalFailure(t *testing.T) {
	acceptable := (time.Hour * 24 * 28).Milliseconds()
	condition := Condition("[CERTIFICATE_EXPIRATION] > " + strconv.FormatInt(acceptable, 10))
	result := &Result{CertificateExpiration: time.Hour * 24 * 14}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[CERTIFICATE_EXPIRATION] (1209600000) > 2419200000"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithCertificateExpirationGreaterThanDuration(t *testing.T) {
	condition := Condition("[CERTIFICATE_EXPIRATION] > 12h")
	result := &Result{CertificateExpiration: 24 * time.Hour}
	condition.evaluate(result)
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	expectedConditionDisplayed := "[CERTIFICATE_EXPIRATION] > 12h"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

func TestCondition_evaluateWithCertificateExpirationGreaterThanDurationFailure(t *testing.T) {
	condition := Condition("[CERTIFICATE_EXPIRATION] > 48h")
	result := &Result{CertificateExpiration: 24 * time.Hour}
	condition.evaluate(result)
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	expectedConditionDisplayed := "[CERTIFICATE_EXPIRATION] (86400000) > 48h (172800000)"
	if result.ConditionResults[0].Condition != expectedConditionDisplayed {
		t.Errorf("Condition '%s' should have resolved to '%s', got '%s'", condition, expectedConditionDisplayed, result.ConditionResults[0].Condition)
	}
}

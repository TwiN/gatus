package endpoint

import (
	"strings"
	"testing"
)

func BenchmarkCondition_evaluateWithBodyStringAny(b *testing.B) {
	condition := Condition("[BODY].name == any(john.doe, jane.doe)")
	for n := 0; n < b.N; n++ {
		result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithBodyStringAnyFailure(b *testing.B) {
	condition := Condition("[BODY].name == any(john.doe, jane.doe)")
	for n := 0; n < b.N; n++ {
		result := &Result{Body: []byte("{\"name\": \"bob.doe\"}")}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithBodyString(b *testing.B) {
	condition := Condition("[BODY].name == john.doe")
	for n := 0; n < b.N; n++ {
		result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithBodyStringFailure(b *testing.B) {
	condition := Condition("[BODY].name == john.doe")
	for n := 0; n < b.N; n++ {
		result := &Result{Body: []byte("{\"name\": \"bob.doe\"}")}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithBodyStringFailureInvalidPath(b *testing.B) {
	condition := Condition("[BODY].user.name == bob.doe")
	for n := 0; n < b.N; n++ {
		result := &Result{Body: []byte("{\"name\": \"bob.doe\"}")}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithBodyStringLen(b *testing.B) {
	condition := Condition("len([BODY].name) == 8")
	for n := 0; n < b.N; n++ {
		result := &Result{Body: []byte("{\"name\": \"john.doe\"}")}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithBodyStringLenFailure(b *testing.B) {
	condition := Condition("len([BODY].name) == 8")
	for n := 0; n < b.N; n++ {
		result := &Result{Body: []byte("{\"name\": \"bob.doe\"}")}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithStatus(b *testing.B) {
	condition := Condition("[STATUS] == 200")
	for n := 0; n < b.N; n++ {
		result := &Result{HTTPStatus: 200}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithStatusFailure(b *testing.B) {
	condition := Condition("[STATUS] == 200")
	for n := 0; n < b.N; n++ {
		result := &Result{HTTPStatus: 400}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

func BenchmarkCondition_evaluateWithNestedWildcard(b *testing.B) {
	var body strings.Builder
	body.WriteString(`{"networks":[`)
	for i := 0; i < 30; i++ {
		if i > 0 {
			body.WriteString(",")
		}
		body.WriteString(`{"name":"network","uplinks":[`)
		for j := 0; j < 30; j++ {
			if j > 0 {
				body.WriteString(",")
			}
			body.WriteString(`{"interface":"wan1","status":"active","ip":"10.0.0.1"}`)
		}
		body.WriteString("]}")
	}
	body.WriteString("]}")
	condition := Condition("[BODY].networks[*].uplinks[*].status == active")
	for n := 0; n < b.N; n++ {
		result := &Result{Body: []byte(body.String())}
		condition.evaluate(result, false, false, nil)
	}
	b.ReportAllocs()
}

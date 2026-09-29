package endpoint

import (
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

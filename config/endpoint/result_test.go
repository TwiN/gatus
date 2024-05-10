package endpoint

import (
	"testing"
)

func TestResult_AddError(t *testing.T) {
	result := &Result{}
	result.AddError("potato")
	if len(result.Errors) != 1 {
		t.Error("should've had 1 error")
	}
	result.AddError("potato")
	if len(result.Errors) != 1 {
		t.Error("should've still had 1 error, because a duplicate error was added")
	}
	result.AddError("tomato")
	if len(result.Errors) != 2 {
		t.Error("should've had 2 error")
	}
}

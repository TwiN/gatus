package core

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

func TestResult_determineSeverityStatus(t *testing.T) {
	scenarios := []struct {
		result                                       *Result
		actualSeverityStatus, expectedSeverityStatus SeverityStatus
	}{
		{result: &Result{Severity: Severity{}}, actualSeverityStatus: Critical, expectedSeverityStatus: SeverityStatus(4)},
		{result: &Result{Severity: Severity{Low: true}}, actualSeverityStatus: Low, expectedSeverityStatus: SeverityStatus(1)},
		{result: &Result{Severity: Severity{Medium: true}}, actualSeverityStatus: Medium, expectedSeverityStatus: SeverityStatus(2)},
		{result: &Result{Severity: Severity{Low: true, Medium: true}}, actualSeverityStatus: Medium, expectedSeverityStatus: SeverityStatus(2)},
		{result: &Result{Severity: Severity{High: true}}, actualSeverityStatus: High, expectedSeverityStatus: SeverityStatus(3)},
		{result: &Result{Severity: Severity{Low: true, Medium: true, High: true}}, actualSeverityStatus: High, expectedSeverityStatus: SeverityStatus(3)},
		{result: &Result{Severity: Severity{Critical: true}}, actualSeverityStatus: Critical, expectedSeverityStatus: SeverityStatus(4)},
		{result: &Result{Severity: Severity{Low: true, Medium: true, High: true, Critical: true}}, actualSeverityStatus: Critical, expectedSeverityStatus: SeverityStatus(4)},
	}

	for _, scenario := range scenarios {
		if scenario.result.determineSeverityStatus() != scenario.expectedSeverityStatus {
			t.Errorf("expected %v, got %v", scenario.expectedSeverityStatus, scenario.actualSeverityStatus)
		}
	}
}

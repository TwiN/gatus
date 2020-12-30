package storage

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"gorm.io/gorm"
)

func TestStorage_ConvertFromStorageWithNoErrorsOrConditionResults(t *testing.T) {
	httpStatus := 200
	hostname := "hostname"
	duration := time.Second * 2
	var errors []evaluationError
	var conditionResults []conditionResult
	success := true
	timestamp := time.Now()

	storageResult := result{
		Model: gorm.Model{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{Valid: false},
		},
		HTTPStatus:       httpStatus,
		Hostname:         hostname,
		Duration:         duration,
		Errors:           errors,
		ConditionResults: conditionResults,
		Success:          success,
		Timestamp:        timestamp,
		ServiceID:        1,
	}

	out := ConvertFromStorage(storageResult)

	if out.HTTPStatus != httpStatus {
		t.Errorf("HTTPStatus should've been %d, but was %d", httpStatus, out.HTTPStatus)
	}
	if out.Hostname != hostname {
		t.Errorf("Hostname should've been %s, but was %s", hostname, out.Hostname)
	}
	if out.Duration != duration {
		t.Errorf("Duration should've been %v, but was %v", duration, out.Duration)
	}
	if len(out.Errors) != 0 {
		t.Errorf("Errors should've been empty slice, but was %v", out.Errors)
	}
	if len(out.ConditionResults) != 0 {
		t.Errorf("ConditionResults should've been empty slice, but was %v", out.ConditionResults)
	}
	if out.Success != success {
		t.Errorf("Success should've been %t, but was %t", success, out.Success)
	}
	if out.Timestamp != timestamp {
		t.Errorf("Timestamp should've been %v, but was %v", timestamp, out.Timestamp)
	}
}

func TestStorage_ConvertFromStorageWithErrors(t *testing.T) {
	errors := []evaluationError{
		{
			Model:    gorm.Model{ID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(), DeletedAt: gorm.DeletedAt{Valid: false}},
			Message:  "Error1",
			ResultID: 1,
		},
		{
			Model:    gorm.Model{ID: 2, CreatedAt: time.Now(), UpdatedAt: time.Now(), DeletedAt: gorm.DeletedAt{Valid: false}},
			Message:  "Error2",
			ResultID: 2,
		},
	}

	storageResult := result{
		Model: gorm.Model{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{Valid: false},
		},
		HTTPStatus:       200,
		Hostname:         "hostname'",
		Duration:         time.Second * 2,
		Errors:           errors,
		ConditionResults: nil,
		Success:          true,
		Timestamp:        time.Now(),
		ServiceID:        1,
	}

	out := ConvertFromStorage(storageResult)

	if len(out.Errors) != 2 {
		t.Fatalf("Errors converted from storage should have been of length 2, but was %d", len(out.Errors))
	}

	for i, err := range out.Errors {
		if err != errors[i].Message {
			t.Errorf("Error at index %d should've had message %s, but was %s", i, errors[i].Message, err)
		}
	}
}

func TestStorage_ConvertFromStorageWithConditionResults(t *testing.T) {
	crs := []conditionResult{
		{
			Model:     gorm.Model{ID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(), DeletedAt: gorm.DeletedAt{Valid: false}},
			Condition: "Condition1",
			Success:   false,
			ResultID:  1,
		},
		{
			Model:     gorm.Model{ID: 2, CreatedAt: time.Now(), UpdatedAt: time.Now(), DeletedAt: gorm.DeletedAt{Valid: false}},
			Condition: "Condition2",
			Success:   true,
			ResultID:  2,
		},
	}

	storageResult := result{
		Model: gorm.Model{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{Valid: false},
		},
		HTTPStatus:       200,
		Hostname:         "hostname'",
		Duration:         time.Second * 2,
		Errors:           nil,
		ConditionResults: crs,
		Success:          true,
		Timestamp:        time.Now(),
		ServiceID:        1,
	}

	out := ConvertFromStorage(storageResult)

	if len(out.ConditionResults) != 2 {
		t.Fatalf("ConditionResults converted from storage should have been of length 2, but was %d", len(out.Errors))
	}

	for i, cr := range out.ConditionResults {
		if cr.Condition != crs[i].Condition {
			t.Errorf("ConditionResult at index %d should've had condition %s, but was %s", i, crs[i].Condition, cr.Condition)
		}
		if cr.Success != crs[i].Success {
			t.Errorf("ConditionResult at index %d should've had success as %t, but was %t", i, crs[i].Success, cr.Success)
		}
	}
}

func TestStorage_ConvertToStorageWithNoErrorsOrConditionResults(t *testing.T) {
	httpStatus := 200
	dnsrCode := "DNSRCode"
	var body []byte
	hostname := "hostname"
	ip := "ip"
	connected := true
	duration := time.Second * 2
	var errors []string
	var conditionResults []*core.ConditionResult
	success := true
	timestamp := time.Now()
	certExpiration := time.Second * 2

	coreResult := core.Result{
		HTTPStatus:            httpStatus,
		DNSRCode:              dnsrCode,
		Body:                  body,
		Hostname:              hostname,
		IP:                    ip,
		Connected:             connected,
		Duration:              duration,
		Errors:                errors,
		ConditionResults:      conditionResults,
		Success:               success,
		Timestamp:             timestamp,
		CertificateExpiration: certExpiration,
	}

	out := ConvertToStorage(coreResult)

	if out.HTTPStatus != httpStatus {
		t.Errorf("HTTPStatus should've been %d, but was %d", httpStatus, out.HTTPStatus)
	}
	if out.Hostname != hostname {
		t.Errorf("Hostname should've been %s, but was %s", hostname, out.Hostname)
	}
	if out.Duration != duration {
		t.Errorf("Duration should've been %v, but was %v", duration, out.Duration)
	}
	if len(out.Errors) != 0 {
		t.Errorf("Errors should've been empty slice, but was %v", out.Errors)
	}
	if len(out.ConditionResults) != 0 {
		t.Errorf("ConditionResults should've been empty slice, but was %v", out.ConditionResults)
	}
	if out.Success != success {
		t.Errorf("Success should've been %t, but was %t", success, out.Success)
	}
	if out.Timestamp != timestamp {
		t.Errorf("Timestamp should've been %v, but was %v", timestamp, out.Timestamp)
	}
}

func TestStorage_ConvertToStorageWithErrors(t *testing.T) {
	errors := []string{
		"Error1",
		"Error2",
	}

	coreResult := core.Result{
		HTTPStatus:            200,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "hostname",
		IP:                    "ip",
		Connected:             true,
		Duration:              time.Second * 2,
		Errors:                errors,
		ConditionResults:      nil,
		Success:               true,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}

	out := ConvertToStorage(coreResult)

	if len(out.Errors) != 2 {
		t.Fatalf("Errors converted from storage should have been of length 2, but was %d", len(out.Errors))
	}

	for i, err := range out.Errors {
		if err.Message != errors[i] {
			t.Errorf("Error at index %d should've had message %s, but was %s", i, errors[i], err.Message)
		}
	}
}

func TestStorage_ConvertToStorageWithConditionResults(t *testing.T) {
	crs := []*core.ConditionResult{
		{
			Condition: "Condition1",
			Success:   false,
		},
		{
			Condition: "Condition2",
			Success:   true,
		},
	}

	coreResult := core.Result{
		HTTPStatus:            200,
		DNSRCode:              "DNSRCode",
		Body:                  nil,
		Hostname:              "hostname",
		IP:                    "ip",
		Connected:             true,
		Duration:              time.Second * 2,
		Errors:                nil,
		ConditionResults:      crs,
		Success:               true,
		Timestamp:             time.Now(),
		CertificateExpiration: time.Second * 2,
	}

	out := ConvertToStorage(coreResult)

	if len(out.ConditionResults) != 2 {
		t.Fatalf("ConditionResults converted from storage should have been of length 2, but was %d", len(out.Errors))
	}

	for i, cr := range out.ConditionResults {
		if cr.Condition != crs[i].Condition {
			t.Errorf("ConditionResult at index %d should've had condition %s, but was %s", i, crs[i].Condition, cr.Condition)
		}
		if cr.Success != crs[i].Success {
			t.Errorf("ConditionResult at index %d should've had success as %t, but was %t", i, crs[i].Success, cr.Success)
		}
	}
}

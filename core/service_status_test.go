package core

import (
	"testing"
	"time"
)

func TestNewServiceStatus(t *testing.T) {
	service := &Service{Name: "name", Group: "group"}
	serviceStatus := NewServiceStatus(service.Key(), service.Group, service.Name)
	if serviceStatus.Name != service.Name {
		t.Errorf("expected %s, got %s", service.Name, serviceStatus.Name)
	}
	if serviceStatus.Group != service.Group {
		t.Errorf("expected %s, got %s", service.Group, serviceStatus.Group)
	}
	if serviceStatus.Key != "group_name" {
		t.Errorf("expected %s, got %s", "group_name", serviceStatus.Key)
	}
}

func TestServiceStatus_AddResult(t *testing.T) {
	service := &Service{Name: "name", Group: "group"}
	serviceStatus := NewServiceStatus(service.Key(), service.Group, service.Name)
	for i := 0; i < MaximumNumberOfResults+10; i++ {
		serviceStatus.AddResult(&Result{Timestamp: time.Now()})
	}
	if len(serviceStatus.Results) != MaximumNumberOfResults {
		t.Errorf("expected serviceStatus.Results to not exceed a length of %d", MaximumNumberOfResults)
	}
}

func TestServiceStatus_WithResultPagination(t *testing.T) {
	service := &Service{Name: "name", Group: "group"}
	serviceStatus := NewServiceStatus(service.Key(), service.Group, service.Name)
	for i := 0; i < 25; i++ {
		serviceStatus.AddResult(&Result{Timestamp: time.Now()})
	}
	if len(serviceStatus.WithResultPagination(1, 1).Results) != 1 {
		t.Errorf("expected to have 1 result")
	}
	if len(serviceStatus.WithResultPagination(5, 0).Results) != 0 {
		t.Errorf("expected to have 0 results")
	}
	if len(serviceStatus.WithResultPagination(-1, 20).Results) != 0 {
		t.Errorf("expected to have 0 result, because the page was invalid")
	}
	if len(serviceStatus.WithResultPagination(1, -1).Results) != 0 {
		t.Errorf("expected to have 0 result, because the page size was invalid")
	}
	if len(serviceStatus.WithResultPagination(1, 10).Results) != 10 {
		t.Errorf("expected to have 10 results, because given a page size of 10, page 1 should have 10 elements")
	}
	if len(serviceStatus.WithResultPagination(2, 10).Results) != 10 {
		t.Errorf("expected to have 10 results, because given a page size of 10, page 2 should have 10 elements")
	}
	if len(serviceStatus.WithResultPagination(3, 10).Results) != 5 {
		t.Errorf("expected to have 5 results, because given a page size of 10, page 3 should have 5 elements")
	}
	if len(serviceStatus.WithResultPagination(4, 10).Results) != 0 {
		t.Errorf("expected to have 0 results, because given a page size of 10, page 4 should have 0 elements")
	}
	if len(serviceStatus.WithResultPagination(1, 50).Results) != 25 {
		t.Errorf("expected to have 25 results, because there's only 25 results")
	}
}

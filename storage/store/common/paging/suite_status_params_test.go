package paging

import (
	"testing"
)

func TestNewSuiteStatusParams(t *testing.T) {
	params := NewSuiteStatusParams()
	if params == nil {
		t.Fatal("NewSuiteStatusParams should not return nil")
	}
	if params.Page != 1 {
		t.Errorf("expected default Page to be 1, got %d", params.Page)
	}
	if params.PageSize != 20 {
		t.Errorf("expected default PageSize to be 20, got %d", params.PageSize)
	}
}

func TestSuiteStatusParams_WithPagination(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedPage int
		expectedSize int
	}{
		{
			name:         "valid pagination",
			page:         2,
			pageSize:     50,
			expectedPage: 2,
			expectedSize: 50,
		},
		{
			name:         "zero page",
			page:         0,
			pageSize:     10,
			expectedPage: 0,
			expectedSize: 10,
		},
		{
			name:         "negative page",
			page:         -1,
			pageSize:     20,
			expectedPage: -1,
			expectedSize: 20,
		},
		{
			name:         "zero page size",
			page:         1,
			pageSize:     0,
			expectedPage: 1,
			expectedSize: 0,
		},
		{
			name:         "negative page size",
			page:         1,
			pageSize:     -10,
			expectedPage: 1,
			expectedSize: -10,
		},
		{
			name:         "large values",
			page:         1000,
			pageSize:     10000,
			expectedPage: 1000,
			expectedSize: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewSuiteStatusParams().WithPagination(tt.page, tt.pageSize)
			if params.Page != tt.expectedPage {
				t.Errorf("expected Page to be %d, got %d", tt.expectedPage, params.Page)
			}
			if params.PageSize != tt.expectedSize {
				t.Errorf("expected PageSize to be %d, got %d", tt.expectedSize, params.PageSize)
			}
		})
	}
}

func TestSuiteStatusParams_ChainedMethods(t *testing.T) {
	params := NewSuiteStatusParams().
		WithPagination(3, 100)
	
	if params.Page != 3 {
		t.Errorf("expected Page to be 3, got %d", params.Page)
	}
	if params.PageSize != 100 {
		t.Errorf("expected PageSize to be 100, got %d", params.PageSize)
	}
}

func TestSuiteStatusParams_OverwritePagination(t *testing.T) {
	params := NewSuiteStatusParams()
	
	// Set initial pagination
	params.WithPagination(2, 50)
	if params.Page != 2 || params.PageSize != 50 {
		t.Error("initial pagination not set correctly")
	}
	
	// Overwrite pagination
	params.WithPagination(5, 200)
	if params.Page != 5 {
		t.Errorf("expected Page to be overwritten to 5, got %d", params.Page)
	}
	if params.PageSize != 200 {
		t.Errorf("expected PageSize to be overwritten to 200, got %d", params.PageSize)
	}
}

func TestSuiteStatusParams_ReturnsSelf(t *testing.T) {
	params := NewSuiteStatusParams()
	
	// Verify WithPagination returns the same instance
	result := params.WithPagination(1, 20)
	if result != params {
		t.Error("WithPagination should return the same instance for method chaining")
	}
}
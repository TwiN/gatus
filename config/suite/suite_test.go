package suite

import (
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/gontext"
)

func TestSuite_ValidateAndSetDefaults(t *testing.T) {
	tests := []struct {
		name    string
		suite   *Suite
		wantErr bool
	}{
		{
			name: "valid-suite",
			suite: &Suite{
				Name: "test-suite",
				Endpoints: []*endpoint.Endpoint{
					{
						Name: "endpoint1",
						URL:  "https://example.org",
						Conditions: []endpoint.Condition{
							endpoint.Condition("[STATUS] == 200"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "suite-without-name",
			suite: &Suite{
				Endpoints: []*endpoint.Endpoint{
					{
						Name: "endpoint1",
						URL:  "https://example.org",
						Conditions: []endpoint.Condition{
							endpoint.Condition("[STATUS] == 200"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "suite-without-endpoints",
			suite: &Suite{
				Name:      "test-suite",
				Endpoints: []*endpoint.Endpoint{},
			},
			wantErr: true,
		},
		{
			name: "suite-with-duplicate-endpoint-names",
			suite: &Suite{
				Name: "test-suite",
				Endpoints: []*endpoint.Endpoint{
					{
						Name: "duplicate",
						URL:  "https://example.org",
						Conditions: []endpoint.Condition{
							endpoint.Condition("[STATUS] == 200"),
						},
					},
					{
						Name: "duplicate",
						URL:  "https://example.com",
						Conditions: []endpoint.Condition{
							endpoint.Condition("[STATUS] == 200"),
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.suite.ValidateAndSetDefaults()
			if (err != nil) != tt.wantErr {
				t.Errorf("Suite.ValidateAndSetDefaults() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Check defaults were set
			if err == nil {
				if tt.suite.Interval == 0 {
					t.Errorf("Expected Interval to be set to default, got 0")
				}
				if tt.suite.Timeout == 0 {
					t.Errorf("Expected Timeout to be set to default, got 0")
				}
			}
		})
	}
}

func TestSuite_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled *bool
		want    bool
	}{
		{
			name:    "nil-defaults-to-true",
			enabled: nil,
			want:    true,
		},
		{
			name:    "explicitly-enabled",
			enabled: boolPtr(true),
			want:    true,
		},
		{
			name:    "explicitly-disabled",
			enabled: boolPtr(false),
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Suite{Enabled: tt.enabled}
			if got := s.IsEnabled(); got != tt.want {
				t.Errorf("Suite.IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSuite_Key(t *testing.T) {
	tests := []struct {
		name  string
		suite *Suite
		want  string
	}{
		{
			name: "with-group",
			suite: &Suite{
				Name:  "test-suite",
				Group: "test-group",
			},
			want: "test-group_test-suite",
		},
		{
			name: "without-group",
			suite: &Suite{
				Name:  "test-suite",
				Group: "",
			},
			want: "_test-suite",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.suite.Key(); got != tt.want {
				t.Errorf("Suite.Key() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSuite_DefaultValues(t *testing.T) {
	s := &Suite{
		Name: "test",
		Endpoints: []*endpoint.Endpoint{
			{
				Name: "endpoint1",
				URL:  "https://example.org",
				Conditions: []endpoint.Condition{
					endpoint.Condition("[STATUS] == 200"),
				},
			},
		},
	}
	err := s.ValidateAndSetDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Interval != DefaultInterval {
		t.Errorf("Expected Interval to be %v, got %v", DefaultInterval, s.Interval)
	}
	if s.Timeout != DefaultTimeout {
		t.Errorf("Expected Timeout to be %v, got %v", DefaultTimeout, s.Timeout)
	}
	if s.InitialContext == nil {
		t.Error("Expected InitialContext to be initialized, got nil")
	}
}

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}

func TestStoreResultValues(t *testing.T) {
	ctx := gontext.New(nil)
	// Create a mock result
	result := &endpoint.Result{
		HTTPStatus: 200,
		IP:         "192.168.1.1",
		Duration:   100 * time.Millisecond,
		Body:       []byte(`{"status": "OK", "value": 42}`),
		Connected:  true,
	}
	// Define store mappings
	mappings := map[string]string{
		"response_code": "[STATUS]",
		"server_ip":     "[IP]",
		"response_time": "[RESPONSE_TIME]",
		"status":        "[BODY].status",
		"value":         "[BODY].value",
		"connected":     "[CONNECTED]",
	}
	// Store values
	stored, err := StoreResultValues(ctx, mappings, result)
	if err != nil {
		t.Fatalf("Unexpected error storing values: %v", err)
	}
	// Verify stored values
	if stored["response_code"] != int64(200) {
		t.Errorf("Expected response_code=200, got %v", stored["response_code"])
	}
	if stored["server_ip"] != "192.168.1.1" {
		t.Errorf("Expected server_ip=192.168.1.1, got %v", stored["server_ip"])
	}
	if stored["status"] != "OK" {
		t.Errorf("Expected status=OK, got %v", stored["status"])
	}
	if stored["value"] != int64(42) { // Now parsed as int64 for whole numbers
		t.Errorf("Expected value=42, got %v", stored["value"])
	}
	if stored["connected"] != true {
		t.Errorf("Expected connected=true, got %v", stored["connected"])
	}
	// Verify values are in context
	val, err := ctx.Get("status")
	if err != nil || val != "OK" {
		t.Errorf("Expected status=OK in context, got %v, err=%v", val, err)
	}
}

func TestStoreResultValuesWithInvalidPath(t *testing.T) {
	ctx := gontext.New(map[string]interface{}{})
	result := &endpoint.Result{
		HTTPStatus: 200,
		Body:       []byte(`{"data": {"name": "john"}}`),
	}
	// Define store mappings with invalid paths
	mappings := map[string]string{
		"valid_status":   "[STATUS]",
		"invalid_token":  "[BODY].accessToken",     // This path doesn't exist
		"invalid_nested": "[BODY].user.id.invalid", // This nested path doesn't exist
	}
	// Store values - should return error for invalid paths
	stored, err := StoreResultValues(ctx, mappings, result)
	if err == nil {
		t.Fatal("Expected error when storing invalid paths, got nil")
	}
	// Check that the error message contains information about the invalid paths
	if !strings.Contains(err.Error(), "invalid_token") {
		t.Errorf("Error should mention invalid_token, got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Errorf("Error should mention 'invalid path', got: %v", err)
	}
	// Verify that valid values were still stored
	if stored["valid_status"] != int64(200) {
		t.Errorf("Expected valid_status=200, got %v", stored["valid_status"])
	}
	// Verify that invalid values show error messages in stored map
	if !strings.Contains(stored["invalid_token"].(string), "ERROR") {
		t.Errorf("Expected invalid_token to contain ERROR, got %v", stored["invalid_token"])
	}
	// Verify that invalid values are NOT in context
	_, err = ctx.Get("invalid_token")
	if err == nil {
		t.Error("Invalid token should not be stored in context")
	}
	// Verify that valid value IS in context
	val, err := ctx.Get("valid_status")
	if err != nil || val != int64(200) {
		t.Errorf("Expected valid_status=200 in context, got %v, err=%v", val, err)
	}
}

func TestSuite_ExecuteWithAlwaysRunEndpoints(t *testing.T) {
	suite := &Suite{
		Name: "test-suite",
		Endpoints: []*endpoint.Endpoint{
			{
				Name: "create-resource",
				URL:  "https://example.org",
				Conditions: []endpoint.Condition{
					endpoint.Condition("[STATUS] == 200"),
				},
				Store: map[string]string{
					"created_id": "[BODY]",
				},
			},
			{
				Name: "failing-endpoint",
				URL:  "https://example.org",
				Conditions: []endpoint.Condition{
					endpoint.Condition("[STATUS] != 200"), // This will fail
				},
			},
			{
				Name: "cleanup-resource",
				URL:  "https://example.org",
				Conditions: []endpoint.Condition{
					endpoint.Condition("[STATUS] == 200"),
				},
				AlwaysRun: true,
			},
		},
	}
	if err := suite.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("suite validation failed: %v", err)
	}
	result := suite.Execute()
	if result.Success {
		t.Error("expected suite to fail due to middle endpoint failure")
	}
	if len(result.EndpointResults) != 3 {
		t.Errorf("expected 3 endpoint results, got %d", len(result.EndpointResults))
	}
	if result.EndpointResults[0].Name != "create-resource" {
		t.Errorf("expected first endpoint to be 'create-resource', got '%s'", result.EndpointResults[0].Name)
	}
	if result.EndpointResults[1].Name != "failing-endpoint" {
		t.Errorf("expected second endpoint to be 'failing-endpoint', got '%s'", result.EndpointResults[1].Name)
	}
	if result.EndpointResults[1].Success {
		t.Error("expected failing-endpoint to fail")
	}
	if result.EndpointResults[2].Name != "cleanup-resource" {
		t.Errorf("expected third endpoint to be 'cleanup-resource', got '%s'", result.EndpointResults[2].Name)
	}
	if !result.EndpointResults[2].Success {
		t.Error("expected cleanup endpoint to succeed")
	}
}

func TestSuite_ExecuteWithoutAlwaysRunEndpoints(t *testing.T) {
	suite := &Suite{
		Name: "test-suite",
		Endpoints: []*endpoint.Endpoint{
			{
				Name: "create-resource",
				URL:  "https://example.org",
				Conditions: []endpoint.Condition{
					endpoint.Condition("[STATUS] == 200"),
				},
			},
			{
				Name: "failing-endpoint",
				URL:  "https://example.org",
				Conditions: []endpoint.Condition{
					endpoint.Condition("[STATUS] != 200"), // This will fail
				},
			},
			{
				Name: "skipped-endpoint",
				URL:  "https://example.org",
				Conditions: []endpoint.Condition{
					endpoint.Condition("[STATUS] == 200"),
				},
			},
		},
	}
	if err := suite.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("suite validation failed: %v", err)
	}
	result := suite.Execute()
	if result.Success {
		t.Error("expected suite to fail due to middle endpoint failure")
	}
	if len(result.EndpointResults) != 2 {
		t.Errorf("expected 2 endpoint results (execution should stop after failure), got %d", len(result.EndpointResults))
	}
	if result.EndpointResults[0].Name != "create-resource" {
		t.Errorf("expected first endpoint to be 'create-resource', got '%s'", result.EndpointResults[0].Name)
	}
	if result.EndpointResults[1].Name != "failing-endpoint" {
		t.Errorf("expected second endpoint to be 'failing-endpoint', got '%s'", result.EndpointResults[1].Name)
	}
}

func TestResult_AddError(t *testing.T) {
	result := &Result{
		Name:      "test-suite",
		Timestamp: time.Now(),
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors initially, got %d", len(result.Errors))
	}
	result.AddError("first error")
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error after AddError, got %d", len(result.Errors))
	}
	if result.Errors[0] != "first error" {
		t.Errorf("Expected 'first error', got '%s'", result.Errors[0])
	}
	result.AddError("second error")
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors after second AddError, got %d", len(result.Errors))
	}
	if result.Errors[1] != "second error" {
		t.Errorf("Expected 'second error', got '%s'", result.Errors[1])
	}
}

func TestResult_CalculateSuccess(t *testing.T) {
	tests := []struct {
		name            string
		endpointResults []*endpoint.Result
		errors          []string
		expectedSuccess bool
	}{
		{
			name:            "no-endpoints-no-errors",
			endpointResults: []*endpoint.Result{},
			errors:          []string{},
			expectedSuccess: true,
		},
		{
			name: "all-endpoints-successful-no-errors",
			endpointResults: []*endpoint.Result{
				{Success: true},
				{Success: true},
			},
			errors:          []string{},
			expectedSuccess: true,
		},
		{
			name: "second-endpoint-failed-no-errors",
			endpointResults: []*endpoint.Result{
				{Success: true},
				{Success: false},
			},
			errors:          []string{},
			expectedSuccess: false,
		},
		{
			name: "first-endpoint-failed-no-errors",
			endpointResults: []*endpoint.Result{
				{Success: false},
				{Success: true},
			},
			errors:          []string{},
			expectedSuccess: false,
		},
		{
			name: "all-endpoints-successful-with-errors",
			endpointResults: []*endpoint.Result{
				{Success: true},
				{Success: true},
			},
			errors:          []string{"suite level error"},
			expectedSuccess: false,
		},
		{
			name: "endpoint-failed-and-errors",
			endpointResults: []*endpoint.Result{
				{Success: true},
				{Success: false},
			},
			errors:          []string{"suite level error"},
			expectedSuccess: false,
		},
		{
			name:            "no-endpoints-with-errors",
			endpointResults: []*endpoint.Result{},
			errors:          []string{"configuration error"},
			expectedSuccess: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Name:            "test-suite",
				Timestamp:       time.Now(),
				EndpointResults: tt.endpointResults,
				Errors:          tt.errors,
			}
			result.CalculateSuccess()
			if result.Success != tt.expectedSuccess {
				t.Errorf("Expected success=%v, got %v", tt.expectedSuccess, result.Success)
			}
		})
	}
}

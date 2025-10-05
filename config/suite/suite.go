package suite

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/gontext"
	"github.com/TwiN/gatus/v5/config/key"
)

var (
	// ErrSuiteWithNoName is the error returned when a suite has no name
	ErrSuiteWithNoName = errors.New("suite must have a name")

	// ErrSuiteWithNoEndpoints is the error returned when a suite has no endpoints
	ErrSuiteWithNoEndpoints = errors.New("suite must have at least one endpoint")

	// ErrSuiteWithDuplicateEndpointNames is the error returned when a suite has duplicate endpoint names
	ErrSuiteWithDuplicateEndpointNames = errors.New("suite cannot have duplicate endpoint names")

	// ErrSuiteWithInvalidTimeout is the error returned when a suite has an invalid timeout
	ErrSuiteWithInvalidTimeout = errors.New("suite timeout must be positive")

	// DefaultInterval is the default interval for suite execution
	DefaultInterval = 10 * time.Minute

	// DefaultTimeout is the default timeout for suite execution
	DefaultTimeout = 5 * time.Minute
)

// Suite is a collection of endpoints that are executed sequentially with shared context
type Suite struct {
	// Name of the suite. Must be unique.
	Name string `yaml:"name"`

	// Group the suite belongs to. Used for grouping multiple suites together.
	Group string `yaml:"group,omitempty"`

	// Enabled defines whether the suite is enabled
	Enabled *bool `yaml:"enabled,omitempty"`

	// Interval is the duration to wait between suite executions
	Interval time.Duration `yaml:"interval,omitempty"`

	// Timeout is the maximum duration for the entire suite execution
	Timeout time.Duration `yaml:"timeout,omitempty"`

	// InitialContext holds initial values that can be referenced by endpoints
	InitialContext map[string]interface{} `yaml:"context,omitempty"`

	// Endpoints in the suite (executed sequentially)
	Endpoints []*endpoint.Endpoint `yaml:"endpoints"`
}

// IsEnabled returns whether the suite is enabled
func (s *Suite) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// Key returns a unique key for the suite
func (s *Suite) Key() string {
	return key.ConvertGroupAndNameToKey(s.Group, s.Name)
}

// ValidateAndSetDefaults validates the suite configuration and sets default values
func (s *Suite) ValidateAndSetDefaults() error {
	// Validate name
	if len(s.Name) == 0 {
		return ErrSuiteWithNoName
	}
	// Validate endpoints
	if len(s.Endpoints) == 0 {
		return ErrSuiteWithNoEndpoints
	}
	// Check for duplicate endpoint names
	endpointNames := make(map[string]bool)
	for _, ep := range s.Endpoints {
		if endpointNames[ep.Name] {
			return fmt.Errorf("%w: duplicate endpoint name '%s'", ErrSuiteWithDuplicateEndpointNames, ep.Name)
		}
		endpointNames[ep.Name] = true
		// Suite endpoints inherit the group from the suite
		ep.Group = s.Group
		// Validate each endpoint
		if err := ep.ValidateAndSetDefaults(); err != nil {
			return fmt.Errorf("invalid endpoint '%s': %w", ep.Name, err)
		}
	}
	// Set default interval
	if s.Interval == 0 {
		s.Interval = DefaultInterval
	}
	// Set default timeout
	if s.Timeout == 0 {
		s.Timeout = DefaultTimeout
	}
	// Validate timeout
	if s.Timeout < 0 {
		return ErrSuiteWithInvalidTimeout
	}
	// Initialize context if nil
	if s.InitialContext == nil {
		s.InitialContext = make(map[string]interface{})
	}
	return nil
}

// Execute executes all endpoints in the suite sequentially with context sharing
func (s *Suite) Execute() *Result {
	start := time.Now()
	// Initialize context from suite configuration
	ctx := gontext.New(s.InitialContext)
	// Create suite result
	result := &Result{
		Name:            s.Name,
		Group:           s.Group,
		Success:         true,
		Timestamp:       start,
		EndpointResults: make([]*endpoint.Result, 0, len(s.Endpoints)),
	}
	// Set up timeout for the entire suite execution
	timeoutChan := time.After(s.Timeout)
	// Execute each endpoint sequentially
	suiteHasFailed := false
	for _, ep := range s.Endpoints {
		// Skip non-always-run endpoints if suite has already failed
		if suiteHasFailed && !ep.AlwaysRun {
			continue
		}
		// Check timeout
		select {
		case <-timeoutChan:
			result.AddError(fmt.Sprintf("suite execution timed out after %v", s.Timeout))
			result.Success = false
			break
		default:
		}
		// Execute endpoint with context
		epStartTime := time.Now()
		epResult := ep.EvaluateHealthWithContext(ctx)
		epDuration := time.Since(epStartTime)
		// Set endpoint name, timestamp, and duration on the result
		epResult.Name = ep.Name
		epResult.Timestamp = epStartTime
		epResult.Duration = epDuration
		// Store values from the endpoint result if configured (always store, even on failure)
		if ep.Store != nil {
			_, err := StoreResultValues(ctx, ep.Store, epResult)
			if err != nil {
				epResult.AddError(fmt.Sprintf("failed to store values: %v", err))
			}
		}
		result.EndpointResults = append(result.EndpointResults, epResult)
		// Mark suite as failed on any endpoint failure
		if !epResult.Success {
			result.Success = false
			suiteHasFailed = true
		}
	}
	result.Context = ctx.GetAll()
	result.Duration = time.Since(start)
	result.CalculateSuccess()
	return result
}

// StoreResultValues extracts values from an endpoint result and stores them in the gontext
func StoreResultValues(ctx *gontext.Gontext, mappings map[string]string, result *endpoint.Result) (map[string]interface{}, error) {
	if mappings == nil || len(mappings) == 0 {
		return nil, nil
	}
	storedValues := make(map[string]interface{})
	var extractionErrors []string
	for contextKey, placeholder := range mappings {
		value, err := extractValueForStorage(placeholder, result)
		if err != nil {
			// Continue storing other values even if one fails
			extractionErrors = append(extractionErrors, fmt.Sprintf("%s: %v", contextKey, err))
			storedValues[contextKey] = fmt.Sprintf("ERROR: %v", err)
			continue
		}
		if err := ctx.Set(contextKey, value); err != nil {
			return storedValues, fmt.Errorf("failed to store %s: %w", contextKey, err)
		}
		storedValues[contextKey] = value
	}
	// Return an error if any values failed to extract
	if len(extractionErrors) > 0 {
		return storedValues, fmt.Errorf("failed to extract values: %s", strings.Join(extractionErrors, "; "))
	}
	return storedValues, nil
}

// extractValueForStorage extracts a value from an endpoint result for storage in context
func extractValueForStorage(placeholder string, result *endpoint.Result) (interface{}, error) {
	// Use the unified ResolvePlaceholder function (no context needed for extraction)
	resolved, err := endpoint.ResolvePlaceholder(placeholder, result, nil)
	if err != nil {
		return nil, err
	}
	// Check if the resolution resulted in an INVALID placeholder
	// This happens when a path doesn't exist (e.g., [BODY].nonexistent)
	if strings.HasSuffix(resolved, " "+endpoint.InvalidConditionElementSuffix) {
		return nil, fmt.Errorf("invalid path: %s", strings.TrimSuffix(resolved, " "+endpoint.InvalidConditionElementSuffix))
	}
	// Try to parse as number or boolean to store as proper types
	// Try int first for whole numbers
	if num, err := strconv.ParseInt(resolved, 10, 64); err == nil {
		return num, nil
	}
	// Then try float for decimals
	if num, err := strconv.ParseFloat(resolved, 64); err == nil {
		return num, nil
	}
	// Then try boolean
	if boolVal, err := strconv.ParseBool(resolved); err == nil {
		return boolVal, nil
	}
	return resolved, nil
}

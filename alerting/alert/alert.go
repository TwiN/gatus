package alert

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/TwiN/logr"
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
	"gopkg.in/yaml.v3"
)

var (
	// ErrAlertWithInvalidDescription is the error with which Gatus will panic if an alert has an invalid character
	ErrAlertWithInvalidDescription         = errors.New("alert description must not have \" or \\")
	ErrAlertWithInvalidDescriptionJSONPath = errors.New("alert description JSONPath is invalid")
)

// Placeholders
const (
	// BodyPlaceholder is a placeholder for the Body of the response
	//
	// Values that could replace the placeholder: {}, {"data":{"name":"john"}}, ...
	BodyPlaceholder = "[BODY]"
)

// Alert is endpoint.Endpoint's alert configuration
type Alert struct {
	// Type of alert (required)
	Type Type `yaml:"type"`

	// Enabled defines whether the alert is enabled
	//
	// Use Alert.IsEnabled() to retrieve the value of this field.
	//
	// This is a pointer, because it is populated by YAML and we need to know whether it was explicitly set to a value
	// or not for provider.ParseWithDefaultAlert to work.
	Enabled *bool `yaml:"enabled,omitempty"`

	// FailureThreshold is the number of failures in a row needed before triggering the alert
	FailureThreshold int `yaml:"failure-threshold"`

	// SuccessThreshold defines how many successful executions must happen in a row before an ongoing incident is marked as resolved
	SuccessThreshold int `yaml:"success-threshold"`

	// Description of the alert. Will be included in the alert sent.
	//
	// This is a pointer, because it is populated by YAML and we need to know whether it was explicitly set to a value
	// or not for provider.ParseWithDefaultAlert to work.
	Description *string `yaml:"description,omitempty"`

	// SendOnResolved defines whether to send a second notification when the issue has been resolved
	//
	// This is a pointer, because it is populated by YAML and we need to know whether it was explicitly set to a value
	// or not for provider.ParseWithDefaultAlert to work. Use Alert.IsSendingOnResolved() for a non-pointer
	SendOnResolved *bool `yaml:"send-on-resolved,omitempty"`

	// ProviderOverride is an optional field that can be used to override the provider's configuration
	// It is freeform so that it can be used for any provider-specific configuration.
	ProviderOverride map[string]any `yaml:"provider-override,omitempty"`

	// ResolveKey is an optional field that is used by some providers (i.e. PagerDuty's dedup_key) to resolve
	// ongoing/triggered incidents
	ResolveKey string `yaml:"-"`

	// Triggered is used to determine whether an alert has been triggered. When an alert is resolved, this value
	// should be set back to false. It is used to prevent the same alert from going out twice.
	//
	// This value should only be modified if the provider.AlertProvider's Send function does not return an error for an
	// alert that hasn't been triggered yet. This doubles as a lazy retry. The reason why this behavior isn't also
	// applied for alerts that are already triggered and has become "healthy" again is to prevent a case where, for
	// some reason, the alert provider always returns errors when trying to send the resolved notification
	// (SendOnResolved).
	Triggered bool `yaml:"-"`
}

// ExtractStringFromResponseBodyByJsonPath extracts a string from the response body by JSON path
func ExtractStringFromResponseBodyByJsonPath(jsonPath string, body []byte) (string, error) {
	// Parse JSON string to interface{}
	parser := oj.Parser{}
	data, err := parser.Parse(body)
	if err != nil {
		return "", fmt.Errorf("[alert.ExtractStringFromResponseBodyByJsonPath] Failed to parse JSON: %w", err)
	}
	// Compile JSONPath expression
	expr, err := jp.ParseString(jsonPath)
	if err != nil {
		return "", fmt.Errorf("[alert.ExtractStringFromResponseBodyByJsonPath] Invalid JSONPath: %w", err)
	}
	// Apply JSONPath
	result := expr.Get(data)
	// Convert result to string
	if len(result) == 0 {
		return "", nil // No matches found
	}
	// Single match — return the first one
	if len(result) == 1 {
		return fmt.Sprintf("%v", result[0]), nil
	}
	// Multiple matches — join with commas
	var parts []string
	for _, v := range result {
		parts = append(parts, fmt.Sprintf("%v", v))
	}
	return strings.Join(parts, ", "), nil
}

// ValidateAndSetDefaults validates the alert's configuration and sets the default value of fields that have one
func (alert *Alert) ValidateAndSetDefaults() error {
	if alert.FailureThreshold <= 0 {
		alert.FailureThreshold = 3
	}
	if alert.SuccessThreshold <= 0 {
		alert.SuccessThreshold = 2
	}
	description := alert.GetDescription(nil)
	if strings.ContainsAny(description, "\"\\") {
		return ErrAlertWithInvalidDescription
	}
	if alert.Description != nil && strings.Contains(description, BodyPlaceholder) && description != BodyPlaceholder {
		_, err := jp.ParseString(strings.TrimPrefix(strings.TrimPrefix(description, BodyPlaceholder), "."))
		if err != nil {
			return ErrAlertWithInvalidDescriptionJSONPath
		}
	}
	return nil
}

// GetDescription retrieves the description of the alert
func (alert *Alert) GetDescription(body []byte) string {
	if alert.Description == nil {
		return ""
	}
	if strings.Contains(*alert.Description, BodyPlaceholder) && body != nil {
		if *alert.Description == BodyPlaceholder {
			return string(body)
		} else {
			str, err := ExtractStringFromResponseBodyByJsonPath(strings.TrimPrefix(strings.TrimPrefix(*alert.Description, BodyPlaceholder), "."), body)
			if err != nil {
				logr.Debug(err.Error())
			}
			return str
		}
	}
	return *alert.Description
}

// IsEnabled returns whether an alert is enabled or not
// Returns true if not set
func (alert *Alert) IsEnabled() bool {
	if alert.Enabled == nil {
		return true
	}
	return *alert.Enabled
}

// IsSendingOnResolved returns whether an alert is sending on resolve or not
func (alert *Alert) IsSendingOnResolved() bool {
	if alert.SendOnResolved == nil {
		return false
	}
	return *alert.SendOnResolved
}

// Checksum returns a checksum of the alert
// Used to determine which persisted triggered alert should be deleted on application start
func (alert *Alert) Checksum() string {
	hash := sha256.New()
	hash.Write([]byte(string(alert.Type) + "_" +
		strconv.FormatBool(alert.IsEnabled()) + "_" +
		strconv.FormatBool(alert.IsSendingOnResolved()) + "_" +
		strconv.Itoa(alert.SuccessThreshold) + "_" +
		strconv.Itoa(alert.FailureThreshold) + "_" +
		alert.GetDescription(nil)),
	)
	return hex.EncodeToString(hash.Sum(nil))
}

func (alert *Alert) ProviderOverrideAsBytes() []byte {
	yamlBytes, err := yaml.Marshal(alert.ProviderOverride)
	if err != nil {
		logr.Warnf("[alert.ProviderOverrideAsBytes] Failed to marshal alert override of type=%s as bytes: %v", alert.Type, err)
	}
	return yamlBytes
}

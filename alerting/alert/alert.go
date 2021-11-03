package alert

// Alert is a core.Endpoint's alert configuration
type Alert struct {
	// Type of alert (required)
	Type Type `yaml:"type"`

	// Enabled defines whether the alert is enabled
	//
	// This is a pointer, because it is populated by YAML and we need to know whether it was explicitly set to a value
	// or not for provider.ParseWithDefaultAlert to work.
	Enabled *bool `yaml:"enabled"`

	// FailureThreshold is the number of failures in a row needed before triggering the alert
	FailureThreshold int `yaml:"failure-threshold"`

	// Description of the alert. Will be included in the alert sent.
	//
	// This is a pointer, because it is populated by YAML and we need to know whether it was explicitly set to a value
	// or not for provider.ParseWithDefaultAlert to work.
	Description *string `yaml:"description"`

	// SendOnResolved defines whether to send a second notification when the issue has been resolved
	//
	// This is a pointer, because it is populated by YAML and we need to know whether it was explicitly set to a value
	// or not for provider.ParseWithDefaultAlert to work. Use Alert.IsSendingOnResolved() for a non-pointer
	SendOnResolved *bool `yaml:"send-on-resolved"`

	// SuccessThreshold defines how many successful executions must happen in a row before an ongoing incident is marked as resolved
	SuccessThreshold int `yaml:"success-threshold"`

	// ResolveKey is an optional field that is used by some providers (i.e. PagerDuty's dedup_key) to resolve
	// ongoing/triggered incidents
	ResolveKey string

	// Triggered is used to determine whether an alert has been triggered. When an alert is resolved, this value
	// should be set back to false. It is used to prevent the same alert from going out twice.
	//
	// This value should only be modified if the provider.AlertProvider's Send function does not return an error for an
	// alert that hasn't been triggered yet. This doubles as a lazy retry. The reason why this behavior isn't also
	// applied for alerts that are already triggered and has become "healthy" again is to prevent a case where, for
	// some reason, the alert provider always returns errors when trying to send the resolved notification
	// (SendOnResolved).
	Triggered bool
}

// GetDescription retrieves the description of the alert
func (alert Alert) GetDescription() string {
	if alert.Description == nil {
		return ""
	}
	return *alert.Description
}

// IsEnabled returns whether an alert is enabled or not
func (alert Alert) IsEnabled() bool {
	if alert.Enabled == nil {
		return false
	}
	return *alert.Enabled
}

// IsSendingOnResolved returns whether an alert is sending on resolve or not
func (alert Alert) IsSendingOnResolved() bool {
	if alert.SendOnResolved == nil {
		return false
	}
	return *alert.SendOnResolved
}

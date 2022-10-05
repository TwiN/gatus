package ntfy

import "testing"

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name:     "valid",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
			expected: true,
		},
		{
			name:     "invalid-url",
			provider: AlertProvider{URL: "", Topic: "example", Priority: 1},
			expected: false,
		},
		{
			name:     "invalid-topic",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "", Priority: 1},
			expected: false,
		},
		{
			name:     "invalid-priority-too-high",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 6},
			expected: false,
		},
		{
			name:     "invalid-priority-too-low",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 0},
			expected: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if scenario.provider.IsValid() != scenario.expected {
				t.Errorf("expected %t, got %t", scenario.expected, scenario.provider.IsValid())
			}
		})
	}
}

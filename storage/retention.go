package storage

// Retention holds everything for the data retention configuration
// Should be later extended with more things
type Retention struct {
	// Uptime Retention Time in Days
	// Default value is here 7 Days
	Days int `yaml:"days"`
}
package buildinfo

import (
	"testing"
)

func TestBuildInfo_Get(t *testing.T) {
	info := Get()
	if info.Version != "dev" {
		t.Errorf("Expected Version 'dev', got '%s'", info.Version)
	}
	if info.CommitHash != "unknown" {
		t.Errorf("Expected CommitHash 'unknown', got '%s'", info.CommitHash)
	}
	if info.Date != "unknown" {
		t.Errorf("Expected BuildDate 'unknown', got '%s'", info.Date)
	}
}

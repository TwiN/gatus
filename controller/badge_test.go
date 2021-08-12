package controller

import (
	"testing"
)

func TestGetBadgeColorFromUptime(t *testing.T) {
	if getBadgeColorFromUptime(1) != "#40cc11" {
		t.Error("expected #40cc11 from an uptime of 1, got", getBadgeColorFromUptime(1))
	}
	if getBadgeColorFromUptime(0.95) != "#94cc11" {
		t.Error("expected #94cc11 from an uptime of 0.95, got", getBadgeColorFromUptime(0.95))
	}
	if getBadgeColorFromUptime(0.9) != "#ccc311" {
		t.Error("expected #c9cc11 from an uptime of 0.9, got", getBadgeColorFromUptime(0.9))
	}
	if getBadgeColorFromUptime(0.85) != "#ccb311" {
		t.Error("expected #ccb311 from an uptime of 0.85, got", getBadgeColorFromUptime(0.85))
	}
	if getBadgeColorFromUptime(0.75) != "#cc8111" {
		t.Error("expected #cc8111 from an uptime of 0.75, got", getBadgeColorFromUptime(0.75))
	}
	if getBadgeColorFromUptime(0.6) != "#cc8111" {
		t.Error("expected #cc8111 from an uptime of 0.6, got", getBadgeColorFromUptime(0.6))
	}
	if getBadgeColorFromUptime(0.25) != "#c7130a" {
		t.Error("expected #c7130a from an uptime of 0.25, got", getBadgeColorFromUptime(0.25))
	}
	if getBadgeColorFromUptime(0) != "#c7130a" {
		t.Error("expected #c7130a from an uptime of 0, got", getBadgeColorFromUptime(0))
	}
}

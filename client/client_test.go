package client

import (
	"os"
	"runtime"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	t.Parallel()
	if success, rtt := Ping("127.0.0.1", &Config{Timeout: 500 * time.Millisecond}); !success {
		t.Error("expected true")
		if rtt == 0 {
			t.Error("Round-trip time returned on success should've higher than 0")
		}
	}
	if success, rtt := Ping("256.256.256.256", &Config{Timeout: 500 * time.Millisecond}); success {
		t.Error("expected false, because the IP is invalid")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	if success, rtt := Ping("192.168.152.153", &Config{Timeout: 500 * time.Millisecond}); success {
		t.Error("expected false, because the IP is valid but the host should be unreachable")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	// Can't perform integration tests (e.g. pinging public targets by single-stacked hostname) here,
	// because ICMP is blocked in the network of GitHub-hosted runners.
	if success, rtt := Ping("127.0.0.1", &Config{Timeout: 500 * time.Millisecond, Network: "ip"}); !success {
		t.Error("expected true")
		if rtt == 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	if success, rtt := Ping("127.0.0.1", &Config{Timeout: 500 * time.Millisecond, Network: "ip6"}); success {
		t.Error("expected false, because the IP isn't an IPv6 address")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
}

func TestShouldRunPingerAsPrivileged_EnvConsistency(t *testing.T) {
	t.Parallel()
	expected := false
	if runtime.GOOS == "windows" {
		expected = true
	} else {
		// non-windows: privileged when effective uid is 0
		if os.Geteuid() == 0 {
			expected = true
		}
	}

	got := ShouldRunPingerAsPrivileged()
	if got != expected {
		t.Fatalf("ShouldRunPingerAsPrivileged returned %v; expected %v (GOOS=%s, euid=%d)", got, expected, runtime.GOOS, os.Geteuid())
	}
}

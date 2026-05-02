package sql

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestLibPQVersionForSupabaseCompatibility verifies that the lib/pq dependency
// is at least v1.11.2, which fixes an issue where lib/pq sends an empty
// "options" startup parameter that Supabase Supavisor rejects with "bad_startup_payload".
//
// This fix is required to resolve: https://github.com/TwiN/gatus/issues/1633
//
// Background:
//   - lib/pq v1.11.0 introduced a Config struct refactor that unconditionally sends
//     an empty "options" parameter during the PostgreSQL startup handshake.
//   - Supabase's Supavisor connection pooler rejects startup messages containing
//     empty parameters, returning :bad_startup_payload and causing an EOF panic.
//   - lib/pq v1.11.2 (commit 1412805) fixes this by omitting empty startup parameters.
//
// See: https://github.com/lib/pq/issues/1259
func TestLibPQVersionForSupabaseCompatibility(t *testing.T) {
	// Find go.mod relative to this test file
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	// Walk up to the module root (where go.mod lives)
	dir := filepath.Dir(testFile)
	var goModPath string
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			goModPath = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find go.mod")
		}
		dir = parent
	}

	data, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("cannot read %s: %v", goModPath, err)
	}

	lines := strings.Split(string(data), "\n")
	var libPQVersion string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "github.com/lib/pq") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				libPQVersion = parts[1]
			}
			break
		}
	}

	if libPQVersion == "" {
		t.Fatal("github.com/lib/pq not found in go.mod")
	}

	if !isLibPQAtLeast(libPQVersion, "v1.11.2") {
		t.Errorf("lib/pq version %s is too old; need at least v1.11.2 for Supabase compatibility. See https://github.com/TwiN/gatus/issues/1633", libPQVersion)
	}
}

// isLibPQAtLeast compares semantic versions of the form "v1.11.1".
// Returns true if `have` >= `need`.
func isLibPQAtLeast(have, need string) bool {
	have = strings.TrimPrefix(have, "v")
	need = strings.TrimPrefix(need, "v")

	haveParts := strings.Split(have, ".")
	needParts := strings.Split(need, ".")

	for i := 0; i < len(needParts); i++ {
		if i >= len(haveParts) {
			return false
		}
		if haveParts[i] > needParts[i] {
			return true
		}
		if haveParts[i] < needParts[i] {
			return false
		}
	}
	return true
}

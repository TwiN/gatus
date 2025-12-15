package buildinfo

import (
	"runtime"
	"testing"
)

func TestBuildInfo_GetDefaults(t *testing.T) {
	info := GetDefault()
	if info.Version != defaultVersion {
		t.Errorf("Expected default Version '%s', got '%s'", defaultVersion, info.Version)
	}
	if info.Revision != defaultRevision {
		t.Errorf("Expected default Revision '%s', got '%s'", defaultRevision, info.Revision)
	}
	if info.Date != defaultDate {
		t.Errorf("Expected default Date '%s', got '%s'", defaultDate, info.Date)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("Expected GoVersion '%s', got '%s'", runtime.Version(), info.GoVersion)
	}
	if info.Dirty != false {
		t.Errorf("Expected Dirty 'false', got '%v'", info.Dirty)
	}
}

func TestBuildInfo_SetLdflags(t *testing.T) {
	t.Run("NoLdflags", func(t *testing.T) {
		info := GetDefault()
		SetLdflags(&info)

		// Since we are not setting ldflags during the test, the values largely remain defaults
		if info.Version != defaultVersion {
			t.Errorf("Expected Version '%s', got '%s'", defaultVersion, info.Version)
		}
		if info.Revision != defaultRevision {
			t.Errorf("Expected Revision '%s', got '%s'", defaultRevision, info.Revision)
		}
		if info.Date != defaultDate {
			t.Errorf("Expected Date '%s', got '%s'", defaultDate, info.Date)
		}
		if info.GoVersion != runtime.Version() {
			t.Errorf("Expected GoVersion '%s', got '%s'", runtime.Version(), info.GoVersion)
		}
		if info.Dirty != false {
			t.Errorf("Expected Dirty 'false', got '%v'", info.Dirty)
		}
	})
	t.Run("WithLdflags", func(t *testing.T) {
		// Simulate ldflags being set
		originalVersion := version
		originalRevision := revision
		originalDate := date
		defer func() {
			version = originalVersion
			revision = originalRevision
			date = originalDate
		}()
		version = "test-version"
		revision = "test-revision"
		date = "test-date"

		info := GetDefault()
		SetLdflags(&info)

		if info.Version != "test-version" {
			t.Errorf("Expected Version 'test-version', got '%s'", info.Version)
		}
		if info.Revision != "test-revision" {
			t.Errorf("Expected Revision 'test-revision', got '%s'", info.Revision)
		}
		if info.Date != "test-date" {
			t.Errorf("Expected Date ''test-date', got '%s'", info.Date)
		}
		if info.GoVersion != runtime.Version() {
			t.Errorf("Expected GoVersion '%s', got '%s'", runtime.Version(), info.GoVersion)
		}
		if info.Dirty != false {
			t.Errorf("Expected Dirty 'false', got '%v'", info.Dirty)
		}
	})
}

func TestBuildInfo_SetEmbedded(t *testing.T) {
	info := GetDefault()
	SetEmbedded(&info)

	if info.Version != defaultVersion {
		t.Errorf("Expected Version '%s', got '%s'", defaultVersion, info.Version)
	}
	if info.Revision != defaultRevision {
		t.Errorf("Expected Revision '%s', got '%s'", defaultRevision, info.Revision)
	}
	if info.Date != defaultDate {
		t.Errorf("Expected Date '%s', got '%s'", defaultDate, info.Date)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("Expected GoVersion '%s', got '%s'", runtime.Version(), info.GoVersion)
	}
	if info.Dirty != false {
		t.Errorf("Expected Dirty 'false', got '%v'", info.Dirty)
	}
}

func TestBuildInfo_SetBuildInfo(t *testing.T) {
	t.Run("TestEmbeddOverwriteLdflags", func(t *testing.T) {
		// Only change one ldflag at a time to test their effects
		originalVersion := version
		defer func() {
			version = originalVersion
		}()
		version = "test-version"

		defaultInfo := GetDefault()
		info := SetBuildInfo(defaultInfo)
		if info.Version != "test-version" {
			t.Errorf("Expected Version 'test-version', got '%s'", info.Version)
		}
		if info.Revision != defaultRevision {
			t.Errorf("Expected Revision '%s', got '%s'", defaultRevision, info.Revision)
		}
		if info.Date != defaultDate {
			t.Errorf("Expected Date '%s', got '%s'", defaultDate, info.Date)
		}
		if info.GoVersion != runtime.Version() {
			t.Errorf("Expected GoVersion '%s', got '%s'", runtime.Version(), info.GoVersion)
		}
		if info.Dirty != false {
			t.Errorf("Expected Dirty 'false', got '%v'", info.Dirty)
		}

		// Change another ldflag
		originalRevision := revision
		defer func() {
			revision = originalRevision
		}()
		revision = "test-rev"

		info = SetBuildInfo(GetDefault())
		if info.Revision != "test-rev" {
			t.Errorf("Expected Revision 'test-rev', got '%s'", info.Revision)
		}

		// Change the last ldflag
		originalDate := date
		defer func() {
			date = originalDate
		}()
		date = "test-date"

		info = SetBuildInfo(defaultInfo)
		if info.Date != "test-date" {
			t.Errorf("Expected Date 'test-date', got '%s'", info.Date)
		}
	})
	t.Run("TestDirtyFlag", func(t *testing.T) {
		info := GetDefault()
		info.Dirty = true // Simulate the embedded info setting Dirty to true
		info = SetBuildInfo(info)
		if info.Dirty != true {
			t.Errorf("Expected Dirty 'true', got '%v'", info.Dirty)
		}
		if info.Revision != defaultRevision+"-dirty" {
			t.Errorf("Expected Revision '%s-dirty', got '%s'", defaultRevision, info.Revision)
		}
	})
}

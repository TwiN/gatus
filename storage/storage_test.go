package storage

import (
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	file := t.TempDir() + "/test.db"
	err := Initialize(&Config{File: file})
	if err != nil {
		t.Fatal("shouldn't have returned an error")
	}
	if cancelFunc == nil {
		t.Error("cancelFunc shouldn't have been nil")
	}
	if ctx == nil {
		t.Error("ctx shouldn't have been nil")
	}
	// Try to initialize it again
	err = Initialize(&Config{File: file})
	if err != nil {
		t.Fatal("shouldn't have returned an error")
	}
	cancelFunc()
}

func TestAutoSave(t *testing.T) {
	file := t.TempDir() + "/test.db"
	if err := Initialize(&Config{File: file}); err != nil {
		t.Fatal("shouldn't have returned an error")
	}
	go autoSave(3*time.Millisecond, ctx)
	time.Sleep(15 * time.Millisecond)
	cancelFunc()
	time.Sleep(5 * time.Millisecond)
}

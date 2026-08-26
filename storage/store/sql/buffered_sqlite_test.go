package sql

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

func TestNewBufferedSQLiteStore(t *testing.T) {
	if _, err := NewBufferedSQLiteStore("", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents); !errors.Is(err, ErrPathNotSpecified) {
		t.Error("expected error due to blank path parameter")
	}
	store, err := NewBufferedSQLiteStore(t.TempDir()+"/TestNewBufferedSQLiteStore.db", true, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	store.Close()
	// An unwritable path must fail loudly at creation rather than on the first flush
	if _, err := NewBufferedSQLiteStore(t.TempDir()+"/does-not-exist/TestNewBufferedSQLiteStore.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents); err == nil {
		t.Error("expected an error, because the parent directory of the path doesn't exist")
	}
}

func TestBufferedStore_SavePersistsToDiskAtomically(t *testing.T) {
	path := t.TempDir() + "/TestBufferedStore_SavePersistsToDiskAtomically.db"
	temporaryPath := path + ".tmp"
	store, err := NewBufferedSQLiteStore(path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	defer store.Close()
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	// Creating the store writes an initial snapshot to probe that the path is writable
	if _, err := os.Stat(path); err != nil {
		t.Fatal("the initial snapshot should've been written on creation, got", err)
	}
	// Deleting the snapshot proves that the inserts below never touch the disk
	if err := os.Remove(path); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if err := store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("inserts shouldn't have touched the disk")
	}
	// Simulate the leftovers of a flush that was interrupted mid-write
	if err := os.WriteFile(temporaryPath, []byte("partial garbage from an interrupted flush"), 0644); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if err := store.Save(); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if _, err := os.Stat(temporaryPath); !os.IsNotExist(err) {
		t.Error("the temporary file should've been renamed over the database file")
	}
	// The snapshot must be a plain sqlite database that the non-buffered store can read
	snapshotStore, err := NewStore("sqlite", path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	defer snapshotStore.Close()
	status, err := snapshotStore.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults).WithEvents(1, storage.DefaultMaximumNumberOfEvents))
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if status == nil || len(status.Results) != 2 || len(status.Events) != 3 {
		t.Error("expected the snapshot to contain 2 results and 3 events, got", status)
	}
}

func TestBufferedStore_ReopenFromSnapshot(t *testing.T) {
	path := t.TempDir() + "/TestBufferedStore_ReopenFromSnapshot.db"
	store, err := NewBufferedSQLiteStore(path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	// Close alone must persist the in-memory database, even without an explicit Save
	store.Close()
	store, err = NewBufferedSQLiteStore(path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	defer store.Close()
	status, err := store.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults).WithEvents(1, storage.DefaultMaximumNumberOfEvents))
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if status == nil || len(status.Results) != 1 {
		t.Fatal("expected the history to survive the restart, got", status)
	}
	// New results written after the restart should stack on top of the restored history
	if err := store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	status, err = store.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults).WithEvents(1, storage.DefaultMaximumNumberOfEvents))
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if status == nil || len(status.Results) != 2 {
		t.Error("expected both the restored and the new result, got", status)
	}
}

func TestBufferedStore_LoadsDatabaseWrittenByUnbufferedStore(t *testing.T) {
	path := t.TempDir() + "/TestBufferedStore_LoadsDatabaseWrittenByUnbufferedStore.db"
	unbufferedStore, err := NewStore("sqlite", path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if err := unbufferedStore.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	unbufferedStore.Close()
	// Leave stale WAL journal siblings behind to make sure the buffered store cleans them up
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.WriteFile(path+suffix, []byte("stale"), 0644); err != nil {
			t.Fatal("shouldn't have returned any error, got", err.Error())
		}
	}
	// Users switching buffered on should keep the history from their existing database
	store, err := NewBufferedSQLiteStore(path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(path + suffix); !os.IsNotExist(err) {
			t.Errorf("expected the stale %s file to have been cleaned up", suffix)
		}
	}
	defer store.Close()
	status, err := store.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults).WithEvents(1, storage.DefaultMaximumNumberOfEvents))
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if status == nil || len(status.Results) != 1 {
		t.Error("expected the history from the unbuffered store to be loaded, got", status)
	}
}

func TestBufferedStore_SurvivesConnectionChurn(t *testing.T) {
	path := t.TempDir() + "/TestBufferedStore_SurvivesConnectionChurn.db"
	store, err := NewBufferedSQLiteStore(path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	defer store.Close()
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	// With no idle pool, database/sql closes the connection after every operation, so each
	// query below runs on a brand new one. This simulates the pool discarding a connection
	// it considers broken; with a plain :memory: DSN, losing the connection would silently
	// wipe the entire in-memory database.
	store.db.SetMaxIdleConns(0)
	status, err := store.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults))
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if status == nil || len(status.Results) != 1 {
		t.Error("expected the in-memory database to survive connection churn, got", status)
	}
}

func TestBufferedStore_FailedSaveDoesNotBreakStore(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "storage")
	path := filepath.Join(directory, "TestBufferedStore_FailedSaveDoesNotBreakStore.db")
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	store, err := NewBufferedSQLiteStore(path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	defer store.Close()
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	// Pull the directory out from under the store, so the flush must fail...
	if err := os.RemoveAll(directory); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if err := store.Save(); err == nil {
		t.Fatal("expected Save to return an error, because the parent directory no longer exists")
	}
	// ...but the store must remain fully usable so that the next tick can retry
	status, err := store.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults))
	if err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if status == nil || len(status.Results) != 1 {
		t.Error("expected the store to remain usable after a failed Save, got", status)
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal("shouldn't have returned any error, got", err.Error())
	}
	if err := store.Save(); err != nil {
		t.Fatal("expected the retried Save to succeed, got", err.Error())
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("expected the database file to exist after the retried Save")
	}
}

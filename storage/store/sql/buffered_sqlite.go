package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TwiN/gocache/v2"
	"github.com/TwiN/logr"
	"modernc.org/sqlite"
)

// errDriverDoesNotSupportRestore should never happen, as gatus always uses modernc.org/sqlite
var errDriverDoesNotSupportRestore = errors.New("sqlite driver does not support the backup api")

// bufferedDatabaseSequence distinguishes the shared in-memory databases of multiple
// buffered stores living in the same process (e.g. tests)
var bufferedDatabaseSequence atomic.Uint64

// BufferedStore is a Store whose sqlite database lives in memory and is only persisted
// to disk by Save and Close; everything else behaves like the regular sqlite Store.
type BufferedStore struct {
	*Store

	// memoryAnchor is an otherwise idle connection that keeps the shared in-memory
	// database alive even if the embedded Store's pool discards its connection
	memoryAnchor *sql.DB

	// flushMutex prevents concurrent flushes from fighting over the same temporary file
	flushMutex sync.Mutex
}

// NewBufferedSQLiteStore initializes an in-memory SQLite database that is only persisted
// to the file at path by Save and Close.
//
// Reads and writes never touch the disk, which spares flash storage (e.g. SD cards on
// single-board computers) at the cost of losing everything written since the last flush
// if the application is not shut down gracefully.
func NewBufferedSQLiteStore(path string, caching bool, maximumNumberOfResults, maximumNumberOfEvents int) (*BufferedStore, error) {
	if len(path) == 0 {
		return nil, ErrPathNotSpecified
	}
	store := &BufferedStore{
		Store: &Store{
			driver:                 "sqlite",
			path:                   path,
			maximumNumberOfResults: maximumNumberOfResults,
			maximumNumberOfEvents:  maximumNumberOfEvents,
		},
	}
	// With a plain :memory: DSN the database would die whenever database/sql replaces its
	// single pooled connection (driver.ErrBadConn), silently wiping the history and letting
	// the next flush overwrite the last good snapshot with an empty database. A named
	// shared-cache database survives as long as one connection stays open: memoryAnchor.
	// foreign_keys goes through the DSN so that replacement connections inherit it too.
	dsn := fmt.Sprintf("file:gatus-buffered-%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", bufferedDatabaseSequence.Add(1))
	var err error
	if store.memoryAnchor, err = sql.Open("sqlite", dsn); err != nil {
		return nil, err
	}
	if err := store.memoryAnchor.Ping(); err != nil {
		_ = store.memoryAnchor.Close()
		return nil, err
	}
	if store.db, err = sql.Open("sqlite", dsn); err != nil {
		_ = store.memoryAnchor.Close()
		return nil, err
	}
	// Same reasoning as NewStore: prevents the driver from running into "database is locked" errors
	store.db.SetMaxOpenConns(1)
	if err := store.db.Ping(); err != nil {
		store.closeDatabaseHandles()
		return nil, err
	}
	if err := store.loadDatabaseFromDisk(); err != nil {
		store.closeDatabaseHandles()
		return nil, fmt.Errorf("failed to load existing database from %s: %w", path, err)
	}
	if err = store.createSchema(); err != nil {
		store.closeDatabaseHandles()
		return nil, err
	}
	// Snapshot right away so that an unwritable path (bad permissions, a single-file bind
	// mount that rename cannot replace) fails at startup, not silently at every tick
	if err := store.writeDatabaseToDisk(); err != nil {
		store.closeDatabaseHandles()
		return nil, fmt.Errorf("failed to write database to %s: %w", path, err)
	}
	if caching {
		store.writeThroughCache = gocache.NewCache().WithMaxSize(writeThroughCacheMaxSize)
	}
	return store, nil
}

// Save persists the in-memory database to disk
func (s *BufferedStore) Save() error {
	return s.writeDatabaseToDisk()
}

// Close persists the in-memory database to disk one last time and closes the store
func (s *BufferedStore) Close() {
	// Best effort: by the time Close is called, there's nobody left to propagate the error to
	if err := s.writeDatabaseToDisk(); err != nil {
		logr.Errorf("[sql.Close] Failed to persist in-memory database to disk: %s", err.Error())
	}
	s.Store.Close()
	// Closed last so that the in-memory database outlives the pool during shutdown
	_ = s.memoryAnchor.Close()
}

// closeDatabaseHandles closes both database handles without flushing anything to disk
func (s *BufferedStore) closeDatabaseHandles() {
	_ = s.db.Close()
	_ = s.memoryAnchor.Close()
}

// loadDatabaseFromDisk copies the database persisted at path into memory using SQLite's
// online backup api. Because the copy is page-for-page, createSchema then takes it
// through the exact same migrations as a database opened directly from disk, including
// one written by an older gatus or by a non-buffered store.
func (s *BufferedStore) loadDatabaseFromDisk() error {
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		logr.Infof("[sql.loadDatabaseFromDisk] Nothing to load, because there is no database at path=%s", s.path)
		return nil
	}
	connection, err := s.db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer connection.Close()
	if err := connection.Raw(func(driverConnection any) error {
		restorer, ok := driverConnection.(interface {
			NewRestore(srcUri string) (*sqlite.Backup, error)
		})
		if !ok {
			return errDriverDoesNotSupportRestore
		}
		restore, err := restorer.NewRestore(s.path)
		if err != nil {
			return err
		}
		if _, err := restore.Step(-1); err != nil {
			_ = restore.Finish()
			return err
		}
		// Despite the field naming inside Backup, in restore mode Finish closes the
		// connection to the file at path, not the pooled connection we're running on
		return restore.Finish()
	}); err != nil {
		return err
	}
	logr.Infof("[sql.loadDatabaseFromDisk] Loaded existing database from path=%s", s.path)
	return nil
}

// writeDatabaseToDisk atomically persists the in-memory database to the file at path:
// snapshot to a temporary file, sync, rename over path. A crash mid-flush can never
// corrupt or truncate the previous good snapshot.
func (s *BufferedStore) writeDatabaseToDisk() error {
	s.flushMutex.Lock()
	defer s.flushMutex.Unlock()
	startTime := time.Now()
	// Same directory as path: a cross-filesystem rename wouldn't be atomic
	temporaryPath := s.path + ".tmp"
	// VACUUM INTO refuses to write to an existing file, so remove what an interrupted
	// flush may have left behind
	if err := os.Remove(temporaryPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if _, err := s.db.Exec("VACUUM INTO ?", temporaryPath); err != nil {
		return err
	}
	// The rename is only durable once the snapshot itself has been synced to disk
	if err := syncFile(temporaryPath); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	// A non-buffered store ran this database in WAL mode; its -wal/-shm files are orphaned
	// once a snapshot is renamed over the main file. Best effort.
	_ = os.Remove(s.path + "-wal")
	_ = os.Remove(s.path + "-shm")
	// Syncing the parent directory persists the rename itself; best effort, because not
	// every platform supports syncing a directory (e.g. Windows)
	if directory, err := os.Open(filepath.Dir(s.path)); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	logr.Debugf("[sql.writeDatabaseToDisk] Persisted in-memory database to path=%s in %s", s.path, time.Since(startTime))
	return nil
}

func syncFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	err = file.Sync()
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

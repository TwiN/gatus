package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/TwinProduction/gatus/storage/store/database"
)

func TestInitialize(t *testing.T) {
	type Scenario struct {
		Name        string
		Cfg         *Config
		ExpectedErr error
	}
	scenarios := []Scenario{
		{
			Name:        "nil",
			Cfg:         nil,
			ExpectedErr: nil,
		},
		{
			Name:        "blank",
			Cfg:         &Config{},
			ExpectedErr: nil,
		},
		{
			Name:        "inmemory-no-file",
			Cfg:         &Config{Type: TypeInMemory},
			ExpectedErr: nil,
		},
		{
			Name:        "inmemory-with-file",
			Cfg:         &Config{Type: TypeInMemory, File: t.TempDir() + "/TestInitialize_inmemory-with-file.db"},
			ExpectedErr: nil,
		},
		{
			Name:        "sqlite-no-file",
			Cfg:         &Config{Type: TypeSQLite},
			ExpectedErr: database.ErrFilePathNotSpecified,
		},
		{
			Name:        "sqlite-with-file",
			Cfg:         &Config{Type: TypeSQLite, File: t.TempDir() + "/TestInitialize_sqlite-with-file.db"},
			ExpectedErr: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := Initialize(scenario.Cfg)
			if err != scenario.ExpectedErr {
				t.Errorf("expected %v, got %v", scenario.ExpectedErr, err)
			}
			if err != nil {
				return
			}
			if cancelFunc == nil {
				t.Error("cancelFunc shouldn't have been nil")
			}
			if ctx == nil {
				t.Error("ctx shouldn't have been nil")
			}
			if provider == nil {
				fmt.Println("wtf?")
			}
			provider.Close()
			// Try to initialize it again
			err = Initialize(scenario.Cfg)
			if err != scenario.ExpectedErr {
				t.Errorf("expected %v, got %v", scenario.ExpectedErr, err)
				return
			}
			provider.Close()
			provider = nil
		})
	}
}

func TestAutoSave(t *testing.T) {
	file := t.TempDir() + "/TestAutoSave.db"
	if err := Initialize(&Config{File: file}); err != nil {
		t.Fatal("shouldn't have returned an error")
	}
	go autoSaveStore(ctx, provider, 3*time.Millisecond)
	time.Sleep(15 * time.Millisecond)
	cancelFunc()
	time.Sleep(10 * time.Millisecond)
}

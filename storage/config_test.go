package storage

import (
	"errors"
	"testing"
	"time"
)

func TestConfig_ValidateAndSetDefaultsWithBufferedStorage(t *testing.T) {
	scenarios := []struct {
		Name                  string
		Cfg                   *Config
		ExpectedErr           error
		ExpectedFlushInterval time.Duration
	}{
		{
			Name:                  "buffered-defaults-flush-interval",
			Cfg:                   &Config{Type: TypeSQLite, Path: "/tmp/gatus.db", Buffered: true},
			ExpectedFlushInterval: DefaultFlushInterval,
		},
		{
			Name:                  "buffered-with-custom-flush-interval",
			Cfg:                   &Config{Type: TypeSQLite, Path: "/tmp/gatus.db", Buffered: true, FlushInterval: time.Hour},
			ExpectedFlushInterval: time.Hour,
		},
		{
			Name:                  "buffered-with-minimum-flush-interval",
			Cfg:                   &Config{Type: TypeSQLite, Path: "/tmp/gatus.db", Buffered: true, FlushInterval: MinimumFlushInterval},
			ExpectedFlushInterval: MinimumFlushInterval,
		},
		{
			Name:        "buffered-with-flush-interval-too-small",
			Cfg:         &Config{Type: TypeSQLite, Path: "/tmp/gatus.db", Buffered: true, FlushInterval: 30 * time.Second},
			ExpectedErr: ErrFlushIntervalTooSmall,
		},
		{
			Name:        "buffered-with-negative-flush-interval",
			Cfg:         &Config{Type: TypeSQLite, Path: "/tmp/gatus.db", Buffered: true, FlushInterval: -time.Minute},
			ExpectedErr: ErrFlushIntervalTooSmall,
		},
		{
			Name:        "flush-interval-without-buffered",
			Cfg:         &Config{Type: TypeSQLite, Path: "/tmp/gatus.db", FlushInterval: time.Hour},
			ExpectedErr: ErrFlushIntervalRequiresBufferedStorage,
		},
		{
			Name:        "buffered-with-memory-type",
			Cfg:         &Config{Type: TypeMemory, Buffered: true},
			ExpectedErr: ErrBufferedStorageRequiresSQLite,
		},
		{
			Name:        "buffered-with-postgres-type",
			Cfg:         &Config{Type: TypePostgres, Path: "postgres://user:password@127.0.0.1:5432/gatus", Buffered: true},
			ExpectedErr: ErrBufferedStorageRequiresSQLite,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Cfg.ValidateAndSetDefaults()
			if !errors.Is(err, scenario.ExpectedErr) {
				t.Errorf("expected error %v, got %v", scenario.ExpectedErr, err)
			}
			if err == nil && scenario.Cfg.FlushInterval != scenario.ExpectedFlushInterval {
				t.Errorf("expected flush interval to be %s, got %s", scenario.ExpectedFlushInterval, scenario.Cfg.FlushInterval)
			}
		})
	}
}

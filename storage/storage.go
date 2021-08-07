package storage

import (
	"context"
	"log"
	"time"

	"github.com/TwinProduction/gatus/storage/store"
	"github.com/TwinProduction/gatus/storage/store/memory"
	"github.com/TwinProduction/gatus/storage/store/sqlite"
)

var (
	provider store.Store

	// initialized keeps track of whether the storage provider was initialized
	// Because store.Store is an interface, a nil check wouldn't be sufficient, so instead of doing reflection
	// every single time Get is called, we'll just lazily keep track of its existence through this variable
	initialized bool

	ctx        context.Context
	cancelFunc context.CancelFunc
)

// Get retrieves the storage provider
func Get() store.Store {
	if !initialized {
		log.Println("[storage][Get] Provider requested before it was initialized, automatically initializing")
		err := Initialize(nil)
		if err != nil {
			panic("failed to automatically initialize store: " + err.Error())
		}
	}
	return provider
}

// Initialize instantiates the storage provider based on the Config provider
func Initialize(cfg *Config) error {
	initialized = true
	var err error
	if cancelFunc != nil {
		// Stop the active autoSaveStore task, if there's already one
		cancelFunc()
	}
	if cfg == nil {
		cfg = &Config{}
	}
	if len(cfg.File) == 0 {
		log.Printf("[storage][Initialize] Creating storage provider with type=%s", cfg.Type)
	} else {
		log.Printf("[storage][Initialize] Creating storage provider with type=%s and file=%s", cfg.Type, cfg.File)
	}
	ctx, cancelFunc = context.WithCancel(context.Background())
	switch cfg.Type {
	case TypeSQLite:
		provider, err = sqlite.NewStore(string(cfg.Type), cfg.File)
		if err != nil {
			return err
		}
	case TypeMemory:
		fallthrough
	default:
		if len(cfg.File) > 0 {
			provider, err = memory.NewStore(cfg.File)
			if err != nil {
				return err
			}
			go autoSaveStore(ctx, provider, 7*time.Minute)
		} else {
			provider, _ = memory.NewStore("")
		}
	}
	return nil
}

// autoSaveStore automatically calls the Save function of the provider at every interval
func autoSaveStore(ctx context.Context, provider store.Store, interval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[storage][autoSaveStore] Stopping active job")
			return
		case <-time.After(interval):
			log.Printf("[storage][autoSaveStore] Saving")
			err := provider.Save()
			if err != nil {
				log.Println("[storage][autoSaveStore] Save failed:", err.Error())
			}
		}
	}
}

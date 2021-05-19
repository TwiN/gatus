package storage

import (
	"context"
	"log"
	"time"

	"github.com/TwinProduction/gatus/storage/store"
	"github.com/TwinProduction/gatus/storage/store/memory"
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
	if cfg == nil || len(cfg.File) == 0 {
		log.Println("[storage][Initialize] Creating storage provider")
		provider, err = memory.NewStore("")
		if err != nil {
			return err
		}
	} else {
		if cancelFunc != nil {
			// Stop the active autoSave task
			cancelFunc()
		}
		ctx, cancelFunc = context.WithCancel(context.Background())
		log.Printf("[storage][Initialize] Creating storage provider with file=%s", cfg.File)
		provider, err = memory.NewStore(cfg.File)
		if err != nil {
			return err
		}
		go autoSave(7*time.Minute, ctx)
	}
	return nil
}

// autoSave automatically calls the SaveFunc function of the provider at every interval
func autoSave(interval time.Duration, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[storage][autoSave] Stopping active job")
			return
		case <-time.After(interval):
			log.Printf("[storage][autoSave] Saving")
			err := provider.Save()
			if err != nil {
				log.Println("[storage][autoSave] Save failed:", err.Error())
			}
		}
	}
}

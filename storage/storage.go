package storage

import (
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
	} else {
		log.Printf("[storage][Initialize] Creating storage provider with file=%s", cfg.File)
		provider, err = memory.NewStore(cfg.File)
		if err != nil {
			return err
		}
		go autoSave(7 * time.Minute)
	}
	return nil
}

// autoSave automatically calls the Save function of the provider at every interval
func autoSave(interval time.Duration) {
	for {
		time.Sleep(interval)
		log.Printf("[storage][autoSave] Saving")
		err := provider.Save()
		if err != nil {
			log.Println("[storage][autoSave] Save failed:", err.Error())
		}
	}
}

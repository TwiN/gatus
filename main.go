package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/controller"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

func main() {
	cfg, err := loadConfiguration()
	if err != nil {
		panic(err)
	}
	initializeStorage(cfg)
	start(cfg)
	// Wait for termination signal
	signalChannel := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		log.Println("Received termination signal, attempting to gracefully shut down")
		stop(cfg)
		save()
		done <- true
	}()
	<-done
	log.Println("Shutting down")
}

func start(cfg *config.Config) {
	go controller.Handle(cfg)
	watchdog.Monitor(cfg)
	go listenToConfigurationFileChanges(cfg)
}

func stop(cfg *config.Config) {
	watchdog.Shutdown(cfg)
	controller.Shutdown()
}

func save() {
	if err := store.Get().Save(); err != nil {
		log.Println("Failed to save storage provider:", err.Error())
	}
}

func loadConfiguration() (*config.Config, error) {
	configPath := os.Getenv("GATUS_CONFIG_PATH")
	// Backwards compatibility
	if len(configPath) == 0 {
		if configPath = os.Getenv("GATUS_CONFIG_FILE"); len(configPath) > 0 {
			log.Println("WARNING: GATUS_CONFIG_FILE is deprecated. Please use GATUS_CONFIG_PATH instead.")
		}
	}
	return config.LoadConfiguration(configPath)
}

// initializeStorage initializes the storage provider
//
// Q: "TwiN, why are you putting this here? Wouldn't it make more sense to have this in the config?!"
// A: Yes. Yes it would make more sense to have it in the config package. But I don't want to import
// the massive SQL dependencies just because I want to import the config, so here we are.
func initializeStorage(cfg *config.Config) {
	err := store.Initialize(cfg.Storage)
	if err != nil {
		panic(err)
	}
	// Remove all EndpointStatus that represent endpoints which no longer exist in the configuration
	var keys []string
	for _, endpoint := range cfg.Endpoints {
		keys = append(keys, endpoint.Key())
	}
	numberOfEndpointStatusesDeleted := store.Get().DeleteAllEndpointStatusesNotInKeys(keys)
	if numberOfEndpointStatusesDeleted > 0 {
		log.Printf("[main][initializeStorage] Deleted %d endpoint statuses because their matching endpoints no longer existed", numberOfEndpointStatusesDeleted)
	}
}

func listenToConfigurationFileChanges(cfg *config.Config) {
	for {
		time.Sleep(30 * time.Second)
		if cfg.HasLoadedConfigurationBeenModified() {
			log.Println("[main][listenToConfigurationFileChanges] Configuration file has been modified")
			stop(cfg)
			time.Sleep(time.Second) // Wait a bit to make sure everything is done.
			save()
			updatedConfig, err := loadConfiguration()
			if err != nil {
				if cfg.SkipInvalidConfigUpdate {
					log.Println("[main][listenToConfigurationFileChanges] Failed to load new configuration:", err.Error())
					log.Println("[main][listenToConfigurationFileChanges] The configuration file was updated, but it is not valid. The old configuration will continue being used.")
					// Update the last file modification time to avoid trying to process the same invalid configuration again
					cfg.UpdateLastFileModTime()
					continue
				} else {
					panic(err)
				}
			}
			store.Get().Close()
			initializeStorage(updatedConfig)
			start(updatedConfig)
			return
		}
	}
}

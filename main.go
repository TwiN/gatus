package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TwiN/gatus/v3/config"
	"github.com/TwiN/gatus/v3/controller"
	"github.com/TwiN/gatus/v3/storage/store"
	"github.com/TwiN/gatus/v3/watchdog"
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
		stop()
		save()
		done <- true
	}()
	<-done
	log.Println("Shutting down")
}

func start(cfg *config.Config) {
	go controller.Handle(cfg.Security, cfg.Web, cfg.UI, cfg.Metrics)
	watchdog.Monitor(cfg)
	go listenToConfigurationFileChanges(cfg)
}

func stop() {
	watchdog.Shutdown()
	controller.Shutdown()
}

func save() {
	if err := store.Get().Save(); err != nil {
		log.Println("Failed to save storage provider:", err.Error())
	}
}

func loadConfiguration() (cfg *config.Config, err error) {
	customConfigFile := os.Getenv("GATUS_CONFIG_FILE")
	if len(customConfigFile) > 0 {
		cfg, err = config.Load(customConfigFile)
	} else {
		cfg, err = config.LoadDefaultConfiguration()
	}
	return
}

// initializeStorage initializes the storage provider
//
// Q: "TwiN, why are you putting this here? Wouldn't it make more sense to have this in the config?!"
// A: Yes. Yes it would make more sense to have it in the config package. But I don't want to import
//    the massive SQL dependencies just because I want to import the config, so here we are.
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
		log.Printf("[config][validateStorageConfig] Deleted %d endpoint statuses because their matching endpoints no longer existed", numberOfEndpointStatusesDeleted)
	}
}

func listenToConfigurationFileChanges(cfg *config.Config) {
	for {
		time.Sleep(30 * time.Second)
		if cfg.HasLoadedConfigurationFileBeenModified() {
			log.Println("[main][listenToConfigurationFileChanges] Configuration file has been modified")
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
			stop()
			start(updatedConfig)
			return
		}
	}
}

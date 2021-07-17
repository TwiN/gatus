package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/controller"
	"github.com/TwinProduction/gatus/storage"
	"github.com/TwinProduction/gatus/watchdog"
)

func main() {
	cfg, err := loadConfiguration()
	if err != nil {
		panic(err)
	}
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
	go controller.Handle(cfg.Security, cfg.Web, cfg.Metrics)
	watchdog.Monitor(cfg)
	go listenToConfigurationFileChanges(cfg)
}

func stop() {
	watchdog.Shutdown()
	controller.Shutdown()
}

func save() {
	err := storage.Get().Save()
	if err != nil {
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

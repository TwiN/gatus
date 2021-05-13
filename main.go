package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Meldiron/gatus/config"
	"github.com/Meldiron/gatus/controller"
	"github.com/Meldiron/gatus/storage"
	"github.com/Meldiron/gatus/watchdog"
)

func main() {
	cfg := loadConfiguration()
	go watchdog.Monitor(cfg)
	go controller.Handle()
	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("Received termination signal, attempting to gracefully shut down")
		controller.Shutdown()
		err := storage.Get().Save()
		if err != nil {
			log.Println("Failed to save storage provider:", err.Error())
		}
		done <- true
	}()
	<-done
	log.Println("Shutting down")
}

func loadConfiguration() *config.Config {
	var err error
	customConfigFile := os.Getenv("GATUS_CONFIG_FILE")
	if len(customConfigFile) > 0 {
		err = config.Load(customConfigFile)
	} else {
		err = config.LoadDefaultConfiguration()
	}
	if err != nil {
		panic(err)
	}
	return config.Get()
}
